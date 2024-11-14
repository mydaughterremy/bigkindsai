import csv
import traceback
from datetime import datetime, timedelta, timezone
import json
from pathlib import Path
import uuid
from log_util import logger, log_elapsed_time
from typing import Any, Dict, List, Optional, Tuple, Union
from argparse import ArgumentParser, Namespace
from refinery.data_fetcher import KindsAPIClient
from refinery.function_block import (
    SEQ_COL_NAME,
    add_column,
    add_sequence_number,
    chunk,
    explode_array,
    load_data_from_disk,
    save_data_to_disk,
    string_to_array,
    transform_field,
)
from indexer.config import parse_conf, IndexingType
from indexer.indexer import Indexer


ap = ArgumentParser(
    description="Data ingestion pipeline for Kinds API",
    epilog="Upstage Co., Ltd. All rights reserved."
)
ap.add_argument("--kinds-api-host", type=str, required=True, help="Kinds API host")
ap.add_argument("--kinds-api-path", type=str, required=True, help="Kinds API path")
ap.add_argument("--kinds-api-key", type=str, required=True, help="Kinds API key")
#ap.add_argument("--local-download-path", type=str, default="/tmp/bigkinds-pipeline", help="Local download path")
ap.add_argument("--local-download-path", type=str, default="./csv", help="Local download path")
ap.add_argument("--indexing-type", type=str, required=True, choices=["WHOLE", "INCREMENTAL"], help="Indexing type")
ap.add_argument("--max-chunk-size", type=int, default=500, help="Max index chunk size")
ap.add_argument("--indexer-endpoint", type=str, required=True, help="Indexer endpoint")
ap.add_argument("--indexer-user", type=str, required=True, help="Indexer user")
ap.add_argument("--indexer-password", type=str, required=True, help="Indexer password")

TENANT_ID = "kinds-ai"
GROUP_ID = "bigkinds"
DOC_TYPE = "news_search"

cur_timezone = timezone(timedelta(hours=9))  # +09:00 (KST)


def send_alert(message: str) -> None:
    # Send alert message if needed
    pass


@log_elapsed_time
def download_file(
    client: KindsAPIClient,
    download_dir: str,
    job_id: uuid.UUID,
    tenant_id: str,
    group_id: str,
    indexing_type: str,
    doc_type: str,
    fields: List[str],
    target_dt: datetime,
) -> str:  # return downloaded file path
    logger.info("Start downloading file")
    daily_data = client.get_daily_data(target_dt, fields)
    logger.info(f"Downloaded {len(daily_data)} documents")

    if len(daily_data) == 0:
        raise RuntimeError("No data fetched")

    filename = f"{job_id}.csv"
    Path(download_dir).mkdir(parents=True, exist_ok=True)
    local_file_path = f"{download_dir}/{filename}"
    logger.info(f"try to write data to local file ({local_file_path})")
    with open(local_file_path, "w", newline='') as csvfile:
        writer = csv.DictWriter(
            csvfile,
            fieldnames=fields,
            delimiter=",",
            quotechar='"',
            quoting=csv.QUOTE_NONNUMERIC
        )

        writer.writeheader()
        for record in daily_data:
            writer.writerow(record)

    logger.info("write csv file done")

    return local_file_path


def pipeline(args: Namespace):
    kinds_api_client = KindsAPIClient(args.kinds_api_host, args.kinds_api_path, args.kinds_api_key)

    logger.info("Start data ingestion pipeline")

    download_dir = args.local_download_path
    tenant_id = TENANT_ID
    group_id = GROUP_ID
    indexing_type = IndexingType[args.indexing_type]
    doc_type = DOC_TYPE
    job_id = uuid.uuid4()
    max_chunk_size = args.max_chunk_size
    indexer_endpoint = args.indexer_endpoint
    indexer_user = args.indexer_user
    indexer_password = args.indexer_password

    start = datetime.now(cur_timezone)
    expected_fields = [
        "news_id",
        "title",
        "content",
        "published_at",
        "enveloped_at",
        "dateline",
        "provider",
        "category",
        "category_incident",
        "hilight",
        "byline",
        "subject_info",
        "subject_info1",
        "subject_info2",
        "subject_info3",
        "subject_info4",
        "provider_link_page",
        "images",
    ]
    target_datetime = start + timedelta(days=-1) + timedelta(hours=-start.hour, minutes=-start.minute, seconds=-start.second, microseconds=-start.microsecond)
    try:
        file_path = download_file(kinds_api_client, download_dir, job_id, tenant_id, group_id, indexing_type, doc_type, expected_fields, target_datetime)
        logger.info(f"raw data is ready: {file_path}")

        data = load_data_from_disk(file_path, "csv", {"delimiter": ",", "quotechar": '"'})

        mapped_data = transform_field(
            data,
            {
                "news_id": "news_id",
                "title": "title",
                "content": "content",
                "published_at": "published_at",
                "enveloped_at": "enveloped_at",
                "dateline": "dateline",
                "provider": "provider",
                "category": "category",
                "category_incident": "category_incident",
                "hilight": "hilight",
                "byline": "byline",
                "provider_link_page": "provider_link_page",
                "images": "images",
                "subject_info": "subject_info",
                "subject_info1": "subject_info1",
                "subject_info2": "subject_info2",
                "subject_info3": "subject_info3",
                "subject_info4": "subject_info4",
            },
        )

        string_to_array_data = string_to_array(
            mapped_data,
            [
                "category",
                "category_incident",
                "subject_info",
                "subject_info1",
                "subject_info2",
                "subject_info3",
                "subject_info4",
            ],
        )

        article_length_added_data = add_column(
            string_to_array_data, "article_length", lambda row: len(row["content"])
        )

        chunk_data = chunk(article_length_added_data, "content", max_chunk_size)

        exploded_data = explode_array(chunk_data, "content")

        seq_num_added_data = add_sequence_number(exploded_data)

        output_file_path = f"{download_dir}/{job_id}-refined.csv"
        metadata_path = save_data_to_disk(
            file_path=output_file_path,
            content=seq_num_added_data,
            content_format="jsonl",
            fields=[
                "news_id",
                "title",
                "content",
                "published_at",
                "enveloped_at",
                "dateline",
                "provider",
                "category",
                "category_incident",
                "hilight",
                "byline",
                "subject_info",
                "subject_info1",
                "subject_info2",
                "subject_info3",
                "subject_info4",
                "provider_link_page",
                "images",
                "article_length",
                SEQ_COL_NAME,
            ],
            attributes={
                "delimiter": ",",
                "quotechar": '"',
                "withheader": True,
            },
        )

        logger.info(f"refined data is ready: {metadata_path}")

        logger.info("Start indexing")
        indexer_config_path = "indexer_config.yml"
        indexer_conf = parse_conf(indexer_config_path)

        indexer_conf.src_file = [metadata_path]
        indexer_conf.group_id = group_id
        indexer_conf.job_id = str(job_id)

        indexer = Indexer(indexer_endpoint, indexer_user, indexer_password, indexer_conf)
        indexer.index_function[indexing_type]()

    except BaseException as be:
        logger.error(be)
        print(traceback.format_exc())
        send_alert(f"Error: {be}")

    end = datetime.now(cur_timezone)
    elapsed_sec = (end - start).total_seconds()
    logger.info(f"End data ingestion pipeline (elapsed: {elapsed_sec:.2f}s)")


if __name__ == "__main__":
    args = ap.parse_args()
    print("hi")
    pipeline(args)
