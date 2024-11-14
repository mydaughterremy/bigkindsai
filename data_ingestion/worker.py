import json
import os
import uuid
from datetime import datetime, timezone
from typing import List, Tuple

import boto3
from botocore.config import Config
from function_block import (
    SEQ_COL_NAME,
    add_column,
    add_sequence_number,
    chunk,
    collection_form,
    explode_array,
    load_data_from_s3,
    publish_sns,
    save_data_to_s3,
    send_slack_message,
    string_to_array,
    transform_field,
)

SRC_BUCKET = os.environ["SRC_BUCKET"]
DEST_BUCKET = os.environ["DEST_BUCKET"]
EVENT_TYPE = os.environ["EVENT_TYPE"]
MAX_CHUNK_SIZE = int(os.environ["MAX_CHUNK_SIZE"])
BROADCAST_TOPIC_ARN = os.environ["BROADCAST_TOPIC_ARN"]

config = Config(retries={"max_attempts": 3})

s3_client = boto3.client("s3", config=config)
glue_client = boto3.client("glue", config=config)
sns_client = boto3.client("sns", config=config)


def lambda_handler(event, context):
    print(f"event: {event}")
    try:
        src_resource = event["src_resource"]
        tenant_id = event["tenant_id"]
        group_id = event["group_id"]
        version = event["version"]
        indexing_type = event["indexing_type"]
        job_id = event["job_id"]

        base_out_prefix = "out/"
        dest_metadata_base_key = (
            f"{base_out_prefix}tenant_id={tenant_id}/type=metadata/version={version}"
        )
        dest_collection_base_key = (
            f"{base_out_prefix}tenant_id={tenant_id}/type={EVENT_TYPE}/version={version}"
        )

        metadata_urls: List[Tuple[str, str]] = []
        collection_urls: List[Tuple[str, str]] = []

        data = load_data_from_s3(
            s3_client,
            SRC_BUCKET,
            src_resource,
            "csv",
            {
                "delimiter": ",",
                "quotechar": '"',
            },
        )

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

        chunk_data = chunk(article_length_added_data, "content", MAX_CHUNK_SIZE)

        exploded_data = explode_array(chunk_data, "content")

        seq_num_added_data = add_sequence_number(exploded_data)

        metadata_url = save_data_to_s3(
            s3_client,
            bucket_name=DEST_BUCKET,
            file_path=f"{dest_metadata_base_key}/{uuid.uuid4()}.csv",
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
        metadata_urls.append(("metadata", metadata_url))

        collection_form_data = collection_form(seq_num_added_data)

        collection_all_url = save_data_to_s3(
            s3_client,
            bucket_name=DEST_BUCKET,
            file_path=f"{dest_collection_base_key}/{uuid.uuid4()}.csv",
            content=collection_form_data,
            content_format="csv",
            fields=[SEQ_COL_NAME, "json"],
            attributes={
                "delimiter": "\t",
                # "quotechar": None, -> no quotechar (QUOTE_NONE)
                "withheader": False,
            },
        )
        collection_urls.append(("collection", collection_all_url))

        publish_sns(
            sns_client,
            metadata_urls,
            collection_urls,
            tenant_id,
            group_id,
            job_id,
            indexing_type,
            BROADCAST_TOPIC_ARN,
        )
    except BaseException as be:
        print(f"Error: {be}")
        send_slack_message(str(be), tenant_id=tenant_id)
        raise be

    return {"statusCode": 200, "body": json.dumps("Hello from Lambda!")}

