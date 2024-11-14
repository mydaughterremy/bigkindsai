import io
import json
import os
import urllib.parse
import urllib.request
from csv import QUOTE_NONNUMERIC, DictReader
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Tuple

import nltk

IngestionDataType = List[Dict[str, Any]]

SEQ_COL_NAME = "__SEQ__"

NLTK_DATA_PATH = "/tmp/nltk_data"


def load_data_from_disk(
    file_path: str,
    content_format: str,
    attributes: Optional[Dict[str, Any]] = None,
    quoting=QUOTE_NONNUMERIC,
) -> IngestionDataType:
    content = []
    with open(file_path, "r") as in_file:
        while True:
            line = in_file.readline()
            if not line:
                break
            content.append(line)

    rows = []
    if content_format == "csv":
        delimiter = attributes["delimiter"] if attributes and "delimiter" in attributes else ","
        quotechar = attributes["quotechar"] if attributes and "quotechar" in attributes else '"'

        # FIXME: newline('\n') in multiline is just removed by default in splitlines()
#        content = content.splitlines()

        reader: DictReader = DictReader(
            content,
            delimiter=delimiter,
            quotechar=quotechar,
            quoting=quoting,
            skipinitialspace=True,
        )
        for row in reader:
            rows.append(row)
    elif content_format == "json":
        raise NotImplementedError()
    elif content_format == "jsonl":
        raise NotImplementedError()
    else:
        raise RuntimeError(f"Unsupported content format: {content_format}")

    return rows


def load_data_from_s3(
    s3_client: "S3.Client",  # noqa: F821  # pyright: ignore[reportUndefinedVariable]
    bucket_name: str,
    file_url: str,
    content_format: str,
    attributes: Optional[Dict[str, Any]] = None,
    quoting=QUOTE_NONNUMERIC,
) -> IngestionDataType:
    url_parsed = urllib.parse.urlparse(file_url)
    url_object = Path(url_parsed.path)
    paths = []
    if url_object.suffix[1:] == content_format:  # assume csv file
        paths += [url_parsed.path[1:]]
    elif url_object.suffix == "":  # assume directory
        res = s3_client.list_objects_v2(Bucket=bucket_name, Prefix=url_parsed.path[1:])
        objs = res["Contents"]
        paths += [obj["Key"] for obj in objs if obj["Key"].endswith(f".{content_format}")]

    rows = []
    for path in paths:
        obj = s3_client.get_object(Bucket=bucket_name, Key=path)

        raw_content = obj["Body"].read().decode("UTF-8")

        if content_format == "csv":
            delimiter = attributes["delimiter"] if attributes and "delimiter" in attributes else ","
            quotechar = attributes["quotechar"] if attributes and "quotechar" in attributes else '"'

            # FIXME: newline('\n') in multiline is just removed by default in splitlines()
            content = raw_content.splitlines()

            reader: DictReader = DictReader(
                content,
                delimiter=delimiter,
                quotechar=quotechar,
                quoting=quoting,
                skipinitialspace=True,
            )
            for row in reader:
                rows.append(row)
        elif content_format == "json":
            raise NotImplementedError()
        elif content_format == "jsonl":
            raise NotImplementedError()
        else:
            raise RuntimeError(f"Unsupported content format: {content_format}")

    return rows


def save_data_to_disk(
    file_path: str,
    content: IngestionDataType,
    content_format: str,
    fields: Optional[List[str]] = None,
    attributes: Optional[Dict[str, Any]] = None,
) -> str:
    if len(content) == 0:
        # TODO: raise error?
        return ""

    keys = fields if fields else set(content[0].keys())

    with io.StringIO() as buf:
        if content_format == "csv":
            import csv

            delimiter = attributes["delimiter"] if attributes and "delimiter" in attributes else ","
            quotechar, quoting = (
                (attributes["quotechar"], csv.QUOTE_NONNUMERIC)
                if attributes and "quotechar" in attributes and len(attributes["quotechar"]) > 0
                else (None, csv.QUOTE_NONE)
            )

            writer = csv.DictWriter(
                buf,
                fieldnames=keys,
                delimiter=delimiter,
                quotechar=quotechar,
                quoting=quoting,
            )

            if attributes and "withheader" in attributes and attributes["withheader"]:
                writer.writeheader()

            new_contents = []
            for c in content:
                new_content = {}
                for key in keys:
                    new_content[key] = c[key]
                new_contents.append(new_content)

            writer.writerows(new_contents)
        elif content_format == "json":
            import json

            json.dump(content, buf, ensure_ascii=False)
        elif content_format == "jsonl":
            import json

            new_contents = []
            for c in content:
                new_content = {}
                for key in keys:
                    new_content[key] = c[key]
                new_contents.append(new_content)

            for c in new_contents:
                buf.write(json.dumps(c, ensure_ascii=False))
                buf.write("\n")
        else:
            raise RuntimeError(f"Unsupported content format: {content_format}")

        with open(file_path, "w") as out_file:
            out_file.write(buf.getvalue())

    return file_path

def save_data_to_s3(
    s3_client: "S3.Client",  # noqa: F821  # pyright: ignore[reportUndefinedVariable]
    bucket_name: str,
    file_path: str,
    content: IngestionDataType,
    content_format: str,
    fields: Optional[List[str]] = None,
    attributes: Optional[Dict[str, Any]] = None,
) -> str:
    if len(content) == 0:
        # TODO: raise error?
        return ""

    keys = fields if fields else set(content[0].keys())

    with io.StringIO() as buf:
        if content_format == "csv":
            import csv

            delimiter = attributes["delimiter"] if attributes and "delimiter" in attributes else ","
            quotechar, quoting = (
                (attributes["quotechar"], csv.QUOTE_NONNUMERIC)
                if attributes and "quotechar" in attributes and len(attributes["quotechar"]) > 0
                else (None, csv.QUOTE_NONE)
            )

            writer = csv.DictWriter(
                buf,
                fieldnames=keys,
                delimiter=delimiter,
                quotechar=quotechar,
                quoting=quoting,
            )

            if attributes and "withheader" in attributes and attributes["withheader"]:
                writer.writeheader()

            new_contents = []
            for c in content:
                new_content = {}
                for key in keys:
                    new_content[key] = c[key]
                new_contents.append(new_content)

            writer.writerows(new_contents)
        elif content_format == "json":
            import json

            json.dump(content, buf, ensure_ascii=False)
        elif content_format == "jsonl":
            import json

            new_contents = []
            for c in content:
                new_content = {}
                for key in keys:
                    new_content[key] = c[key]
                new_contents.append(new_content)

            for c in new_contents:
                buf.write(json.dumps(c, ensure_ascii=False))
                buf.write("\n")
        else:
            raise RuntimeError(f"Unsupported content format: {content_format}")

        result = s3_client.put_object(Bucket=bucket_name, Key=file_path, Body=buf.getvalue())
        if result["ResponseMetadata"]["HTTPStatusCode"] // 100 != 2:
            raise RuntimeError(f"Failed to save file to S3: {result}")

        return f"s3://{bucket_name}/{file_path}"


def transform_field(
    rows: IngestionDataType,
    mapping: Dict[str, str],
) -> IngestionDataType:
    """
    `rows`의 레코드를 사용해 `mapping`에 정의된 대로 필드명을 변환한다.
    :param rows: 레코드 목록
    :param mapping: 필드명 변환 규칙 ({origin: target})
    :return: 필드명 변환 규칙
    """
    for row in rows:
        for key, value in mapping.items():
            tmp = row[key]
            del row[key]
            row[value] = tmp

    return rows


def string_to_array(
    rows: IngestionDataType,
    target_columns: List[str],
) -> IngestionDataType:
    for row in rows:
        for key in target_columns:
            objects = json.loads(row[key].replace("'", '"')) if row[key] else []
            assert isinstance(objects, list)
            for obj in objects:
                assert isinstance(obj, str)
            row[key] = objects

    return rows


def chunk(
    rows: IngestionDataType,
    target_column: str,
    max_chunk_size: int,
) -> IngestionDataType:
    def chunk_size(_chunk: List[str]) -> int:
        return sum([len(sentence) for sentence in _chunk])

    for row in rows:
        sentences = nltk.sent_tokenize(row[target_column])
        chunks = []
        buf = []
        for sentence in sentences:
            if chunk_size(buf) + chunk_size(sentence) + 1 >= max_chunk_size:
                # commit
                chunk = " ".join(buf)
                chunks.append(chunk)
                buf = []

            buf.append(sentence)

        if buf and len(buf) > 0:
            chunk = " ".join(buf)
            chunks.append(chunk)

        row[target_column] = chunks

    return rows


def explode_array(
    rows: IngestionDataType,
    target_column: str,
) -> IngestionDataType:
    exploded = []
    for row in rows:
        assert isinstance(row[target_column], list)

        for value in row[target_column]:
            new_row = row.copy()
            new_row[target_column] = value
            exploded.append(new_row)

    return exploded


def add_sequence_number(
    rows: IngestionDataType,
) -> IngestionDataType:
    for i, row in enumerate(rows):
        row[SEQ_COL_NAME] = i

    return rows


def add_column(
    rows: IngestionDataType, column_name: str, row_handler: Callable[[Dict[str, Any]], Any]
) -> IngestionDataType:
    for row in rows:
        row[column_name] = row_handler(row)

    return rows


def collection_form(
    rows: IngestionDataType,
) -> IngestionDataType:
    data_keys = set(rows[0].keys())

    if SEQ_COL_NAME in data_keys:
        data_keys.discard(SEQ_COL_NAME)

    if len(data_keys) > 1:
        for i, row in enumerate(rows):
            tmp = {}
            for data_key in data_keys:
                tmp[data_key] = row[data_key]
                del row[data_key]
            row[SEQ_COL_NAME] = row[SEQ_COL_NAME]
            row["json"] = json.dumps(tmp, ensure_ascii=False)
    elif len(data_keys) == 1:
        pass
    else:
        print("No data keys found")
        raise RuntimeError("No data keys found")

    return rows


def publish_sns(
    sns_client: "SNS.Client",  # noqa: F821  # pyright: ignore[reportUndefinedVariable]
    metadata_urls: List[Tuple[str, str]],
    collection_urls: List[Tuple[str, str]],
    tenant_id: str,
    group_id: str,
    job_id: str,
    indexing_type: str,
    topic_arn: str,
) -> None:
    msg = {
        "metadata": {
            "tenant_id": tenant_id,
            "group_id": group_id,
            "job": {
                "id": job_id,
                "indexing_type": indexing_type or "WHOLE",
            },
        },
        "type": "COLLECTIONS_CREATED",
        "version": "v1alpha",
        "collections_created": {
            "metadata": [{"name": name, "path": path} for name, path in metadata_urls],
            "collections": [{"name": name, "path": path} for name, path in collection_urls],
        },
    }

    sns_client.publish(
        TopicArn=topic_arn,
        Subject=f"Data Ingestion (tenant_id: {tenant_id}) Complete",
        Message=json.dumps(msg),
    )
