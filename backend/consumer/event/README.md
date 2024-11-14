# 빅카인즈AI API 이벤트 컨슈머

빅카인즈AI API에서 발생하는 이벤트를 수집하여 DB에 알맞게 등록합니다.

## 실행 방법
`go run .`

## 환경변수

| 이름 | 설명 |
| --- | --- |
UPSTAGE_KAFKA_BROKERS | 카프카 브로커 주소
UPSTAGE_KAFKA_TOPIC | 카프카 토픽
UPSTAGE_KAFKA_GROUP | 카프카 컨슈퍼 그룹 이름
UPSTAGE_DATABASE_ENGINE | 데이터베이스 엔진 이름 (오직 mysql 지원)
UPSTAGE_DATABASE_HOST | 데이터베이스 호스트
UPSTAGE_DATABASE_PORT | 데이터베이스 포트
UPSTAGE_DATABASE_USER | 데이터베이스 유저
UPSTAGE_DATABASE_PASSWORD | 데이터베이스 패름워드
UPSTAGE_DATABASE_NAME | 데이터베이스 DB 이름