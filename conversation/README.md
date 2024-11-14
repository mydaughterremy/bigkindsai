# 빅카인즈AI 대화엔진

빅카인즈AI의 RAG 파이프라인을 담당하는 대화엔진입니다.

## 실행 방법
### 코드
`go run .`

### 바이너리
`./conversation_<os>_<arch>` (ex: `conversation_darwin_arm64`) <br>

`--rest-endpoint`: rest API endpoint 설정 (default: :8080)

## 환경변수
.env.sample을 참고하여 .env를 생성하세요. .env는 실행 위치와 동일한 폴더에 있어야 합니다.

| 이름 | 설명 |
| --- | --- |
UPSTAGE_OPENAI_KEY | OpenAI 키
UPSTAGE_SEARCHSERVICE_MSEARCH_ENDPOINT | 검색엔진 엔드포인트, 끝에 /msearch 를 붙여야 합니다.
UPSTAGE_AZURE_ENDPOINT | Azure 엔드포인트
UPSTAGE_AZURE_KEY | Azure 키
UPSTAGE_LLM_MODEL | 사용할 LLM 모델. provider/model_name/retry_count 로 설정 가능하며, 컴마 (,) 로 구분하여 여러 모델을 fallback으로 사용 가능합니다. provider는 openai와 azure가 가능합니다. (예시: openai/gpt-3.5-turbo-1106/1)
KEYWORDS_RELATED_QUERIES_MODE | 연관 키워드 및 검색쿼리 모드 (llm 고정)
FILTER_ARTICLE_LENGTH | 200자 미만 청크를 필터링합니다. (false/true, true 추촌)