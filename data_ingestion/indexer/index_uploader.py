import logging
import os
from typing import Any, Dict, List, Optional, Set

from opensearchpy import OpenSearch
from opensearchpy import helpers as opensearchpy_helpers
from indexer.config import Config
from indexer.util.list_helper import _split_chunks


class IndexUploader:
    client: OpenSearch
    index_template: Dict[Any, Any]
    config: Config
    pid: int

    def __init__(self,
                 endpoint: str,
                 user: str,
                 password: str,
                 config: Config,
                 index_template: Dict[Any, Any]
    ) -> None:
        # define OS client
        self.client = OpenSearch(
            hosts=endpoint,
            http_auth=(user, password),
            http_compress=True,
            #use_ssl=True,
            #verify_certs=True,
            use_ssl=True,
            verify_certs=False,
            ssl_assert_hostname=False,
            ssl_show_warn=False,
            timeout=120,
            retry_on_timeout=True,
            max_retries=3,
        )

        self.config = config
        self.index_template = index_template

    def is_index_exists(self) -> bool:
        return self.client.indices.exists(index=self.config.index_name)

    def get_index_record_count(self) -> int:
        return self.client.count(index=self.config.index_name)["count"]

    # create index, if not exists
    def create_index_if_not_exists(self) -> None:
        # Check if index exists
        if not self.client.indices.exists(index=self.config.index_name):
            self.client.indices.create(index=self.config.index_name, body=self.index_template)

    def create_index(self) -> None:
        # make it sure that index to create does not exist
        if self.client.indices.exists(index=self.config.index_name):
            raise LookupError(f"Index '{self.config.index_name}' already exists")
        self.client.indices.create(index=self.config.index_name, body=self.index_template)

    # bulk add documents to index
    def upload_index(self, docs: List[Any], batch_size: int) -> None:
        # batch insert
        actions = []
        for _, doc in enumerate(docs):
            assert len(list(doc.keys())) == 1
            pid = list(doc.keys())[0]
            actions.append(
                {
                    "_op_type": "index",
                    "_id": pid,
                    "_index": self.config.index_name,
                    "_source": doc[pid],
                }
            )
        for success, info in opensearchpy_helpers.parallel_bulk(
            self.client, actions=actions, thread_count=4, chunk_size=batch_size
        ):
            if not success:
                logging.error(f"A document failed: {info}")

    # refresh indices
    def refresh_indices(self) -> None:
        self.client.indices.refresh(index=self.config.index_name)

    def set_refresh_interval(self, refresh_interval: str) -> None:
        settings = {
            "index": {
                "refresh_interval": refresh_interval,
            }
        }
        self.client.indices.put_settings(index=self.config.index_name, body=settings)

    def get_documents(self, cond: Optional[Dict[str, Any]] = None) -> Any:
        """
        `cond`에 해당하는 문서들을 가져옵니다.
        :param cond: Opensearch QueryDSL의 `match` Query 인자 형식 (`{[key: value]+}` 형태)
        :return: `cond`에 해당하는 문서 목록
        """
        match_cond = cond if cond else {}
        return self.client.search(
            index=self.config.index_name, body={"query": {"match": match_cond}}
        )

    def delete_documents(
        self, cond: Optional[Dict[str, Any]] = None, requests_per_second: int = -1
    ) -> None:
        """
        `cond`에 해당하는 문서들을 삭제합니다.
        :param cond: Opensearch QueryDSL의 `match` Query 인자 형식 (`{[key: value]+}` 형태)
        :param requests_per_second: 한 번에 삭제할 문서의 개수 (=초 당 처리할 요청 수, default: -1 -- 제한 없음)
        """
        match_cond = cond if cond else {}
        self.client.delete_by_query(
            index=self.config.index_name,
            body={"query": {"match": match_cond}},
            requests_per_second=requests_per_second,
        )

    def delete_documents_by_objs(self, key: str, values: Set[Any]) -> None:
        """
        `objs`에 해당하는 문서들을 삭제합니다.
        :param key: 삭제 조건 키
        :param values: 삭제 조건 값들
        """
        if len(values) == 0:
            return

        queries = []
        for value in values:
            queries.append({"match": {key: value}})

        for query_chunk in _split_chunks(queries, 1000):  # maxClauseCount = 1024
            print(f"DocID to delete: {str(query_chunk)[:100]}...")
            self.client.delete_by_query(
                index=self.config.index_name,
                body={"query": {"bool": {"should": query_chunk}}},
                conflicts="proceed",
            )
