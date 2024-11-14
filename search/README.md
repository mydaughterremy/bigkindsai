# 빅카인즈AI 검색엔진

빅카인즈AI 에서 기사 검색을 담당하는 엔진입니다.

## 실행 방법
### 코드
`go run .`<br>

### 바이너리
`./search_<os>_<arch>` (ex: `search_darwin_arm64`) <br>
`-r, --rest-server-endpoint`: rest API 서버 엔드포인트 설정 (default: 0.0.0.0:8080)<br>
`-g, --grpc-server-endpoint`: gRPC 서버 엔드포인트 설정 (default: 0.0.0.0:8081)<br>


## 환경변수
.env.sample을 참고하여 .env를 생성하세요. .env는 실행 위치와 동일한 폴더에 있어야 합니다.

| 이름 | 설명 |
| --- | --- |
| UPSTAGE_OPENSEARCH_ADDRESS | OpenSearch 주소 |
| UPSTAGE_OPENSEARCH_USERNAME | OpenSearch 유저 이름 |
| UPSTAGE_OPENSEARCH_PASSWORD | OpenSearch 비밀번호 |