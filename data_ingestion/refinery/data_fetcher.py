import csv
import json
import logging
import os
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Optional, List, Union, Dict

from log_util import logger
from requests import Request, Session

local_timezone = timezone(timedelta(hours=9))  # +09:00 (KST)


class KindsAPIClient(object):
    dt_format = "%Y-%m-%d"

    def __init__(self, host: str, path: str, key: str):
        self.host = host
        self.path = path
        self.key = key

    @staticmethod
    def build_args(date_start: datetime,
                   date_end: datetime,
                   hilight: int = 0,
                   page_offset: int = 0,
                   page_size: int = 100,
                   fields: List[str] = (),
                   query: Optional[Union[str, Dict[str, str]]] = None,
                   provider: Optional[List[str]] = None,
                   category: Optional[List[str]] = None,
                   category_incident: Optional[List[str]] = None,
                   byline: Optional[str] = None,
                   provider_subject: Optional[List[str]] = None,
                   subject_info: Optional[List[str]] = None,
                   subject_info1: Optional[List[str]] = None,
                   subject_info2: Optional[List[str]] = None,
                   subject_info3: Optional[List[str]] = None,
                   subject_info4: Optional[List[str]] = None,
                   sort: Optional[Union[Dict[str, str], List[str], List[Dict[str, str]]]] = None) -> Dict[str, Any]:

        obj = {}

        assert date_start < date_end, "start date should be earlier than end date"

        obj["published_at"] = {
            "from": date_start.strftime(KindsAPIClient.dt_format),
            "until": date_end.strftime(KindsAPIClient.dt_format),
        }

        if query:
            obj["query"] = query

        if provider:
            obj["provider"] = provider

        if category:
            obj["category"] = category

        if category_incident:
            obj["category_incident"] = category_incident

        if byline:
            obj["byline"] = byline

        if provider_subject:
            obj["provider_subject"] = provider_subject

        if subject_info:
            obj["subject_info"] = subject_info

        if subject_info1:
            obj["subject_info1"] = subject_info1

        if subject_info2:
            obj["subject_info2"] = subject_info2

        if subject_info3:
            obj["subject_info3"] = subject_info3

        if subject_info4:
            obj["subject_info4"] = subject_info4

        if sort:
            obj["sort"] = sort

        obj["hilight"] = hilight
        obj["return_from"] = page_offset
        obj["return_size"] = page_size
        obj["fields"] = fields

        return obj

    def get_daily_data(self, date: datetime, fields: List[str] = ()) -> List[Dict[str, Any]]:
        def call_api(args: Dict[str, Any]):
            def datetime_to_str(dt: datetime) -> str:
                return dt.strftime(KindsAPIClient.dt_format)

            url = self.host + self.path
            logger.info(f"call api: {url}")

            sess = Session()

            _param = {
                "access_key": self.key,
                "argument": args,
            }
            req_body = json.dumps(_param, ensure_ascii=False, default=datetime_to_str).encode("UTF-8")
            logger.info(f"param: {req_body.decode('UTF-8')}")

            headers = {
                "Content-Type": "application/json",
                "User-agent": "Custom Agent/0.0",
            }
            logger.info(f"headers: {headers}")

            req = Request("POST", url, headers=headers, data=req_body)
            prepared = req.prepare()
            res = sess.send(prepared)

            return res.json()

        def get_daily_data_by_page(target_date: datetime,
                                   _page_number: int,
                                   _page_size: int,
                                   target_fields: List[str] = ()) -> Dict[str, Any]:

            args = KindsAPIClient.build_args(
                date_start=target_date,
                date_end=target_date + timedelta(days=1),
                page_offset=_page_number * _page_size,
                page_size=_page_size,
                fields=target_fields,
            )
            result = call_api(args)
            if result["result"] != 0:
                raise RuntimeError(f"API call failed: {result['reason']}")

            return {
                "page_number": _page_number + 1,
                "total_hits": result["return_object"]["total_hits"],
                "page_docs": result["return_object"]["documents"],
                "page_size": len(result["return_object"]["documents"]),
            }

        _DEFAULT_PAGE_SIZE = 10000

        buf = []
        page_number = 0
        page = get_daily_data_by_page(date, page_number, _DEFAULT_PAGE_SIZE, fields)

        total_hits = page["total_hits"]
        page_size = page["page_size"]
        page_number = page["page_number"]
        logger.info(f"""result #{page_number}: {page_size}/{total_hits}""")

        page_docs = page["page_docs"]
        buf += page_docs
        remain_count = total_hits - page_size
        while remain_count > 0:
            page = get_daily_data_by_page(date, page_number, _DEFAULT_PAGE_SIZE, fields)

            page_size = page["page_size"]
            page_number = page["page_number"]
            logger.info(f"""result #{page_number}: {page_size}/{total_hits}""")

            page_docs = page["page_docs"]
            buf += page_docs
            remain_count -= page_size

        return buf


def _parse_datetime(dt: str) -> datetime:
    """
    `dt` could be in the following format:
        YESTERDAY
        TODAY
        YYYY-MM-DD'T'HH:mm:ss.SSS'Z'
        YYYY-MM-DD
        any other format that `datetime.fromisoformat` can parse (ISO 8601 compatible)
    """
    if dt == "YESTERDAY":
        return datetime.now(local_timezone) + timedelta(days=-1)
    elif dt == "TODAY":
        return datetime.now(local_timezone)

    return datetime.fromisoformat(dt)


def lambda_handler(event, context):
    kinds_api_client = KindsAPIClient(HOST, PATH, API_KEY)

    logger.info(f"event: {event}")

    start = datetime.now(timezone.utc)
    try:
        tenant_id = event["tenant_id"]
        indexing_type = event["indexing_type"]
        group_id = event["group_id"]
        file_id = str(uuid.uuid4())
        job_id = str(uuid.uuid4())
        document_type = event["document_type"]
        target_date = _parse_datetime(event["target_datetime"])
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
        logger.info("try to get data from kinds api")
        daily_data = kinds_api_client.get_daily_data(target_date, expected_fields)
        logger.info("done")

        filename = f"{uuid.uuid4()}.csv"
        Path(LOCAL_DOWNLOAD_BASE_DIR).mkdir(parents=True, exist_ok=True)
        local_file_path = f"{LOCAL_DOWNLOAD_BASE_DIR}/{filename}"
        logger.info(f"try to write data to local file ({local_file_path})")
        with open(local_file_path, "w", newline='') as csvfile:
            writer = csv.DictWriter(
                csvfile,
                fieldnames=expected_fields,
                delimiter=",",
                quotechar='"',
                quoting=csv.QUOTE_NONNUMERIC
            )

            writer.writeheader()
            for record in daily_data:
                writer.writerow(record)
        logger.info("done")
    except BaseException as be:
        logger.error(be, exc_info=True)
        send_slack_message(f"Error: {be}")

    end = datetime.now(timezone.utc)
    elapsed_sec = (end - start).total_seconds()
    logger.info(f"Elapsed time: {elapsed_sec:.2f}s")


if __name__ == "__main__":
    manual_event = {
        "tenant_id": "kinds-ai",
        "indexing_type": "WHOLE",
        "group_id": "bigkinds",
        "target_datetime": "2024-01-22",
        "document_type": "news_search",
    }
    context = None
    lambda_handler(manual_event, context)
