import json
import logging
from typing import Any, Callable, Dict, List, Set, TextIO

import requests
from indexer.config import Config, Encoder, Field, IndexingType, VectorField
from indexer.index_uploader import IndexUploader
from indexer.util.list_helper import _split_chunks

INACTIVE_REFRESH_INTERVAL_VALUE = "-1"
DEFAULT_REFRESH_INTERVAL_VALUE = "1s"
COMPRESS_CODEC = "best_compression"
FLUSH_THRESHOLD_SIZE = "2048MB"  # mem * 0.5 * 0.25 = 16GB * 0.5 * 0.25 = 2048MB


def _composite_key(doc: Dict[str, Any], doc_id_fields: List[str]) -> str:
    return "_".join([str(doc[field]) for field in doc_id_fields])


def prepend_inner_key_prefix(name: str) -> str:
    return f"upstage_{name}"


# Indexer
class Indexer:
    config: Config
    index_uploader: IndexUploader

    def __init__(self,
                 endpoint: str,
                 user: str,
                 password: str,
                 config: Config
    ) -> None:
        # set variables
        self.config = config
        index_template = self.create_index_template()
        self.index_uploader = IndexUploader(endpoint, user, password, config, index_template)
        self.index_function: Dict[IndexingType, Callable[[], None]] = {
            IndexingType.WHOLE: self._index_whole,
            IndexingType.INCREMENTAL: self._index_incremental,
        }

    def create_index_template(self) -> Dict[Any, Any]:
        properties = {}
        # inactivate refresh_interval, after indexing, set refresh_interval to default
        self.config.settings["index"]["refresh_interval"] = INACTIVE_REFRESH_INTERVAL_VALUE
        self.config.settings["index"]["codec"] = COMPRESS_CODEC  # set compress codec
        self.config.settings["index"]["translog"] = {
            "flush_threshold_size": FLUSH_THRESHOLD_SIZE
        }  # set flush_threshold_size
        if self.config.fields.text_field:
            for key, field in self.config.fields.text_field.items():
                properties[key] = {
                    "type": field.type,
                }
                if field.analyzer:
                    properties[key]["analyzer"] = field.analyzer
                if field.fields:
                    properties[key]["fields"] = field.fields
                if field.index:
                    properties[key]["index"] = field.index
        if self.config.fields.vector_field:
            self.config.settings["index"]["knn"] = True  # set knn to true
            for key, field in self.config.fields.vector_field.items():
                properties[key] = {
                    "type": "knn_vector",
                    "dimension": field.dimension,
                    "method": field.ann_method,
                }
        index_template = {
            "settings": self.config.settings,
            "mappings": {
                "properties": properties,
            },
        }
        return index_template

    def _merge_source_fields(self, key: str, field: Field, record: Dict[str, Any]) -> str:
        if len(field.src_field) == 1:
            # TODO: consider numeric types (see: https://opensearch.org/docs/latest/search-plugins/sql/datatypes/)
            if field.type == "integer":
                field_value = record[field.src_field[0]]
                _ = int(field_value)  # validation check for integer
                return field_value
            else:
                return str(record[field.src_field[0]])
        elif len(field.src_field) > 1:
            return " ".join([record[f] for f in field.src_field])
        else:
            return str(record[key])

    def _get_document_vector(self, encoder: Encoder, docs: List[str]) -> List[float]:
        if encoder.sentenceTransformer:
            payload = {
                "sentences": docs,
            }
            try:
                response = requests.post(
                    "",
                    json=payload,
                )
                response.raise_for_status()
            except requests.exceptions.RequestException as e:
                logging.error(f"Failed to send apply request: {e}")
                raise e
            return response.json()["embeddings"]
        else:
            raise NotImplementedError

    def _load_index_docs(self, f: TextIO, config: Config) -> List[Dict[str, Any]]:
        index_docs = []
        for _, line in enumerate(f):
            record = json.loads(line)
            doc_id = _composite_key(record, config.doc_id_fields)
            index_doc = {doc_id: {}}
            # handle text fields
            if self.config.fields.text_field:
                for key, field in self.config.fields.text_field.items():
                    index_doc[doc_id][key] = self._merge_source_fields(key, field, record)

            # handle vector fields
            if self.config.fields.vector_field:
                for key, field in self.config.fields.vector_field.items():
                    if field.embedding_method.encoder:
                        index_doc[doc_id][key] = self._merge_source_fields(key, field, record)
                    elif field.embedding_method.file:
                        try:
                            assert len(field.src_field) == 1
                        except AssertionError:
                            raise Exception("src_field should be one when embedding_method is file")
                        index_doc[doc_id][key] = record[field.src_field[0]]

            index_doc[doc_id][prepend_inner_key_prefix("group_id")] = config.group_id

            index_docs.append(index_doc)

        return index_docs

    def _process_index_batch(self, docs: List[Dict[str, Any]], batch_size: int = 2000) -> None:
        doc_chunks = list(_split_chunks(docs, batch_size))

        for chunk_num, doc_chunk in enumerate(doc_chunks):
            logging.info(f"Processing file (chunk #{chunk_num})")

            self._batch_process(doc_chunk, self.config.fields.vector_field)

    def _process_index_docs(self, f: TextIO, config: Config) -> None:
        batch_docs = []
        for idx, line in enumerate(f):
            record = json.loads(line)
            doc_id = _composite_key(record, config.doc_id_fields)
            index_doc = {doc_id: {}}
            # handle text fields
            if self.config.fields.text_field:
                for key, field in self.config.fields.text_field.items():
                    index_doc[doc_id][key] = self._merge_source_fields(key, field, record)

            # handle vector fields
            if self.config.fields.vector_field:
                for key, field in self.config.fields.vector_field.items():
                    if field.embedding_method.encoder:
                        index_doc[doc_id][key] = self._merge_source_fields(key, field, record)
                    elif field.embedding_method.file:
                        try:
                            assert len(field.src_field) == 1
                        except AssertionError:
                            raise Exception("src_field should be one when embedding_method is file")
                        index_doc[doc_id][key] = record[field.src_field[0]]

            index_doc[doc_id][prepend_inner_key_prefix("group_id")] = config.group_id

            batch_docs.append(index_doc)

            # batch process
            if (idx + 1) % config.batch_size == 0:
                logging.info(f"Processing file {idx + 1}")
                self._batch_process(batch_docs, self.config.fields.vector_field)

        if batch_docs:
            self._batch_process(batch_docs, self.config.fields.vector_field)

    def _batch_process(
        self, batch_docs: List[Any], vector_field: Dict[str, VectorField] = None
    ) -> None:
        if vector_field:
            for key, field in vector_field.items():
                logging.info(f"get vector for {key} ({len(batch_docs)} docs)")
                if field.embedding_method.encoder:
                    document_vectors = self._get_document_vector(
                        field.embedding_method.encoder,
                        [list(doc.values())[0][key] for doc in batch_docs],
                    )
                elif field.embedding_method.file:
                    document_vectors = [list(doc.values())[0][key] for doc in batch_docs]
                else:
                    logging.error(f"embedding_method is not defined for {key}")
                    raise Exception(f"embedding_method is not defined for {key}")

                logging.info(f"get vector for {key} done")
                for j, index_doc in enumerate(batch_docs):
                    pid = list(index_doc.keys())[0]
                    index_doc[pid][key] = document_vectors[j]

        logging.info(f"Uploading {len(batch_docs)} docs")
        self.index_uploader.upload_index(batch_docs, 500)
        del batch_docs[:]

    def _index_whole(self) -> None:
        # create index (if not exists)
        logging.info("create index")
        self.index_uploader.create_index()

        # read and indexing
        logging.info("read source file")
        for _, file in enumerate(self.config.src_file):
            with open(file, "r", encoding="utf-8") as f:
                self._process_index_docs(f, self.config)
        logging.info("all index are uploaded")

        # refresh indices
        self.index_uploader.refresh_indices()
        logging.info("refresh indices done")

        # set index refresh_interval to 1s
        self.index_uploader.set_refresh_interval(DEFAULT_REFRESH_INTERVAL_VALUE)
        logging.info("set refresh interval done")

    def _index_incremental(self) -> None:
        logging.info("check if index exists")
        if not self.index_uploader.is_index_exists():
            logging.error(f"Index {self.index_uploader.config.index_name} does not exist:")
            raise RuntimeError("Index does not exist")

        # disable refresh_interval during incremental indexing
        self.index_uploader.set_refresh_interval(INACTIVE_REFRESH_INTERVAL_VALUE)
        logging.info("refresh interval off")

        # read and indexing
        logging.info("read source file")
        new_docs: List[Dict[str, Any]] = []
        for filename in self.config.src_file:
            with open(filename, "r", encoding="utf-8") as f:
                new_docs += self._load_index_docs(f, self.config)
        logging.info("read source file done")

        # get documents where $(dedup_field_value) is already in index
        dedup_field = self.config.dedup_field

        logging.info(f"get documents where `{dedup_field}` is already in index")
        dedup_field_value_set: Set[Any] = set()
        for doc in new_docs:
            obj = list(doc.values())[0]
            dedup_field_value = obj[dedup_field]
            dedup_field_value_set.add(dedup_field_value)

        logging.info(f"delete documents where `{dedup_field}` is already in index")
        self.index_uploader.delete_documents_by_objs(dedup_field, dedup_field_value_set)
        logging.info("delete documents done")

        logging.info("upload new documents")
        self._process_index_batch(new_docs, self.config.batch_size)
        logging.info("upload new documents done")

        # refresh indices
        self.index_uploader.refresh_indices()
        logging.info("refresh index is done")

        # set index refresh_interval to 1s
        self.index_uploader.set_refresh_interval(DEFAULT_REFRESH_INTERVAL_VALUE)
        logging.info(f"refresh interval on ({DEFAULT_REFRESH_INTERVAL_VALUE})")
