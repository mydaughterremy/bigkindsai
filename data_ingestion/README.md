# Data Ingestion

Data Ingestion의 실행 단계는 아래와 같습니다:
* Kinds News API로부터 데이터를 다운로드
* 검색엔진 인덱싱을 위한 데이터 변환
* 검색엔진(AWS OpenSearch)에 인덱싱

## Components

* Pipeline (`pipeline.py`)
  * Data Ingestion 파이프라인을 정의합니다. 개별 모듈을 사용하는 방법이 정의되어 있습니다.
* Refinery (`refinery/`)
  * `data_fetcher.py`: Kinds News API로부터 데이터를 다운로드합니다.
  * `function_block.py`: 데이터 변환을 위한 함수 블록을 정의합니다.
* Indexer (`indexer/`)
  * `indexer.py`: 데이터 인덱싱 방법을 정의합니다. `WHOLE`, `INCREMENTAL` 방식을 지원합니다.
    * `WHOLE`: 전체 데이터를 인덱싱합니다. 검색엔진에 대상 인덱스가 없어야 합니다.
    * `INCREMENTAL`: 기존 인덱스에 증분으로 데이터를 인덱싱합니다. 검색엔진에 대상 인덱스가 있어야 합니다.
  * `index_uploader.py`: AWS OpenSearch의 연결을 담당합니다.

## How to Run
```bash
$ python3 pipeline.py \
    --kinds-api-host [KINDS_API_HOST] \
    --kinds-api-path [KINDS_API_PATH] \
    --kinds-api-key [KINDS_API_KEY] \
    --indexing-type [INDEXING_TYPE: WHOLE/INCREMENTAL] \
    --indexer-user [INDEXER_USERNAME] \
    --indexer-password [INDEXER_PASSWORD] \
    --indexer-endpoint [INDEXER_ENDPOINT]
```

## Data Schema
* Kinds News API 데이터: [OpenAPI 사용자지침서](Kinds_News_API_OpenAPI_사용자지침서_V1.46.pdf)
  * 포맷: CSV
  * 대상 필드
    - news_id
    - title
    - content
    - published_at
    - enveloped_at
    - dateline
    - provider
    - category
    - category_incident
    - hilight
    - byline
    - subject_info
    - subject_info1
    - subject_info2
    - subject_info3
    - subject_info4
    - provider_link_page
    - images
  * 예
    ```csv
    "news_id","title","content","published_at","enveloped_at","dateline","provider","category","category_incident","hilight","byline","subject_info","subject_info1","subject_info2","subject_info3","subject_info4","provider_link_page","images"
    "08200101.20240613151900003","GS건설, 다음 달 '검단아테라자이' 분양","GS건설과 금호건설이 손잡고 검단신도시에 아파트를 공급합니다.

    GS건설 컨소시엄(GS건설㈜, 금호건설㈜)은 다음달 인천시 서구 검단신도시 불로동 484-3 번지 일대에서 '검단아테라자이'를 분양할 예정이라고 밝혔습니다.

    검단아테라자이는 지하 2층에서 지상 최고 25층 6개동으로 전용면적 59~84㎡ 총 709가구 규모로 조성됩니다.

    전용면적별 가구수는 59㎡A 140가구, 59㎡B 23가구, 59㎡C 261가구, 59㎡D 22가구, 59㎡E 22가구, 74㎡ 99가구, 84㎡ 142가구 등 중소형으로 구성됩니다.

    특히, 이 단지는 주택개발 공모리츠사업으로 GS건설 컨소시엄이 주택설계부터 주택사업 인허가, 책임 준공의무까지 맡고 있습니다.

    검단아테라자이는 인천 1호선 연장사업과 다양한 도로망 확충 공사 등으로 향후 교통여건 향상이 기대되는 좋은 입지를 가졌습니다.

    먼저, 인천지하철 1호선 연장 신설역인 검단호수공원역(가칭)이 단지 인근에 들어설 예정으로 인천 주요지역과 서울 도심으로 빠르게 이동이 가능할 전망입니다.

    또 검단~드림로간 도로, 국지도 98호선(도계~마전) 도로, 인천 대곡동~불로지구 연결도로, 금곡동~대곡동간 도로, 검단~경명로간 도로 등 다양한 도로망이 공사 중으로 앞으로 교통여건은 더욱 좋아질 것으로 기대됩니다.

    특히 인근에 있는 인천 대곡동~불로지구 연결도로와 국지도 98호선(도계~마전) 도로가 완공될 경우 김포한강로와 일산대교까지 한 번에 도달이 가능해져 서울의 주요 도심까지 접근성이 크게 향상될 예
    정입니다.

    다양한 생활 인프라도 들어설 예정입니다.

    수변형 상업 특화거리인 커낼콤플렉스와 중심상업지구가 단지에서 가까운 거리에 예정돼 있습니다.

    또 검단신도시를 아우르는 U자형 녹지축 시작점인 근린공원이 단지 인근에 약 9만 3천㎡ 규모의 문화공원이 도보권에 조성될 예정입니다.

    특히 단지 맞은편으로 초등학교와 유치원 예정 부지가 위치해 있어 우수한 교육환경을 갖출 것으로 기대되고 있습니다.

    청약은 인천 및 수도권 거주자 중 청약통장 가입기간 1년 이상 경과하고, 면적과 지역별 예치 기준금액을 충족하면 1순위 자격이 주어지며 유주택자나 세대원도 청약할 수 있습니다.

    검단아테라자이의 견본주택은 경기도 부천시 상동 529-38(부천영상문화단지 내)에 7월 중 개관할 예정이며, 입주는 2027년 2월 예정입니다.<유숙열(ryusy@obs.co.kr)>","2024-06-13T00:00:00.000+09:00","2024-06-13T15:19:00.000+09:00","2024-06-13T15:19:00.000+09:00","OBS","['경제>부동산']","[]","","유숙열","['경제', '산업/기업', 'NEWS']","","","","","http://www.obsnews.co.kr/news/articleView.html?idxno=1444895","/08200101/2024/06/13/08200101.20240613151900003.01.jpg"
    ```
* Refined 데이터
  * 포맷: JSONL 
  * | Field Name         | Type             | Description         |
    |--------------------|------------------|---------------------|
    | news_id            | string           |                     |
    | title              | string           |                     |
    | content            | string           |                     |
    | published_at       | string(datetime) |                     |
    | enveloped_at       | string(datetime) |                     |
    | dateline           | string(datetime) |                     |
    | provider           | string           |                     |
    | category           | List\[string\]   |                     |
    | category_incident  | List\[string\]   |                     |
    | hilight            | string           |                     |
    | byline             | string           |                     |
    | subject_info       | List\[string\]   |                     |
    | subject_info1      | List\[string\]   |                     |
    | subject_info2      | List\[string\]   |                     |
    | subject_info3      | List\[string\]   |                     |
    | subject_info4      | List\[string\]   |                     |
    | provider_link_page | string (URL)     |                     |
    | images             | string (URL)     |                     |
    | article_length     | int              | 기사 본문 길이 (refinery) |
    | \_\_SEQ\_\_        | int              | refinery 결과 일련번호    |
  * 예
      ```json
      {
          "news_id": "08200101.20240613151900003",
          "title": "GS건설, 다음 달 '검단아테라자이' 분양",
          "content": "GS건설과 금호건설이 손잡고 검단신도시에 아파트를 공급합니다.",
          "published_at": "2024-06-13T00:00:00.000+09:00",
          "enveloped_at": "2024-06-13T15:19:00.000+09:00",
          "dateline": "2024-06-13T15:19:00.000+09:00",
          "provider": "OBS",
          "category": [
              "경제>부동산"
          ],
          "category_incident": [],
          "hilight": "",
          "byline": "유숙열",
          "subject_info": [
              "경제",
              "산업/기업",
              "NEWS"
          ],
          "subject_info1": [],
          "subject_info2": [],
          "subject_info3": [],
          "subject_info4": [],
          "provider_link_page": "http://www.obsnews.co.kr/news/articleView.html?idxno=1444895",
          "images": "/08200101/2024/06/13/08200101.20240613151900003.01.jpg",
          "article_length": 1198,
          "__SEQ__": 0
      }
      ```
* Index 데이터
  * 포맷: JSON (기본, 설정 가능 -- https://opensearch.org/docs/2.0/search-plugins/sql/response-formats/)
  * | Field Name         | Type             | Description         |
    |--------------------|------------------|---------------------|
    | news_id            | string           |                     |
    | title              | string           |                     |
    | content            | string           |                     |
    | published_at       | string(datetime) |                     |
    | enveloped_at       | string(datetime) |                     |
    | dateline           | string(datetime) |                     |
    | provider           | string           |                     |
    | category           | List\[string\]   |                     |
    | category_incident  | List\[string\]   |                     |
    | hilight            | string           |                     |
    | byline             | string           |                     |
    | subject_info       | List\[string\]   |                     |
    | subject_info1      | List\[string\]   |                     |
    | subject_info2      | List\[string\]   |                     |
    | subject_info3      | List\[string\]   |                     |
    | subject_info4      | List\[string\]   |                     |
    | provider_link_page | string (URL)     |                     |
    | images             | string (URL)     |                     |
    | article_length     | int              | 기사 본문 길이 (refinery) |
    | upstage_group_id   | string           | 내부 식별용 필드           |

  * 예 (OpenSearch Document)
      ```json
      {
        "_index": "kinds-news-v1",
        "_id": "08200101.20240613151900003_1",
        "_score": 1.0,
        "_source": {
            "news_id": "08200101.20240613151900003",
            "title": "GS건설, 다음 달 '검단아테라자이' 분양",
            "content": "GS건설 컨소시엄(GS건설㈜, 금호건설㈜)은 다음달 인천시 서구 검단신도시 불로동 484-3 번지 일대에서 '검단아테라자이'를 분양할 예정이라고 밝혔습니다.",
            "published_at": "2024-06-13T00:00:00.000+09:00",
            "enveloped_at": "2024-06-13T15:19:00.000+09:00",
            "dateline": "2024-06-13T15:19:00.000+09:00",
            "provider": "OBS",
            "category": "['경제>부동산']",
            "category_incident": "[]",
            "byline": "유숙열",
            "subject_info": "['경제', '산업/기업', 'NEWS']",
            "subject_info1": "[]",
            "subject_info2": "[]",
            "subject_info3": "[]",
            "subject_info4": "[]",
            "provider_link_page": "http://www.obsnews.co.kr/news/articleView.html?idxno=1444895",
            "images": "/08200101/2024/06/13/08200101.20240613151900003.01.jpg",
            "article_length": 1198,
            "upstage_group_id": "bigkinds"
        }
      }
      ```