# BigkindsAI API server

## Overview
한국언론진흥재단 빅카인즈AI API 서버입니다.

## Running the server
두 가지 서버를 함께 구동해야 합니다.
1. 빅카인즈AI API 서버<br>
```go run .```<br>
`--rest-endpoint`: rest API endpoint 설정 (default: :8080)

2. 빅카인즈AI 이벤트 컨슈머<br>
빅카인즈AI에서 발생하는 이벤트를 수신하여 DB에 업데이트합니다.<br>
`consumer/event` 에서 자세한 설명을 보세요.

## 카프카 등 컴포넌트 실행 방법
** Docker compose가 필요합니다. **

```make bootstrap-local-dev-stack```

위 명령 실행 시 모든 필요한 컴포넌트를 실행하고 마이그레이션까지 실행됩니다. 성공 시 바로 서버를 실행하실 수 있습니다.
포트 등 기본 설정은 .env.sample을 참조하세요.

## DB 마이그레이션 방법
```MIGRATION_URL=<mysql uri> make migrate```

## API 스펙
빅카인즈AI_API명세서_V1.1.hwp 참고

## 환경변수 설정
.env.sample를 참고하여 .env를 생성합니다.

|     이름                       |    설명    |
|-------------------------------|-----------|
|UPSTAGE_KAFKA_BROKERS   | 카프카 브로커 주소    |
|UPSTAGE_KAFKA_TOPIC     | 카프카 토픽 이름     |
|UPSTAGE_MYSQL_HOST      | MySQL DB Host     |
|UPSTAGE_MYSQL_PORT      | MySQL DB Port     |
|UPSTAGE_MYSQL_USER      | MySQL 유저         |
|UPSTAGE_MYSQL_PASSWORD  | MySQL 비밀번호      |
|UPSTAGE_MYSQL_DB        | MySQL 데이터베이스    |
UPSTAGE_CONVERSATION_ENGINE_ENDPOINT  | 대화엔진 엔드포인트    |
|UPSTAGE_KINDSAI_KEY             | 빅카인즈AI API 키 |

## 데이터베이스 스키마

### chats
유저가 생성한 대화를 저장하는 테이블입니다.

| field |	type |	설명 |
|-------|--------|------|
| id	| CHAR(36)	| chat의 ID |
| created_at |	DATETIME |	chat 생성 일자 |
| updated_at |	DATETIME |	chat 업데이트 일자 |
| deleted_at |	DATETIME |	chat 삭제 일자 |
| object |	LONGTEXT |	object 이름 (chat으로 고정) |
| title |	LONGTEXT |	chat 제목 |
| session_id |	VARCHAR(191) |	chat을 생성한 session id |

### qas
유저와 빅카인즈AI 사이의 질답을 저장하는 테이블입니다.

field |	type |	설명
|-------|--------|------|
id	| CHAR(36) |	QA (질답) ID
chat_id |	CHAR(36) |	QA가 속하는 chat의 ID
session_id | VARCHAR(36) | 질문자의 세션 ID
job_group |VARCHAR(36) |질문자의 직업
question |TEXT |질문
answer |TEXT |답변
references |JSON |참조 기사
keywords |JSON |키워드
related_queries |JSON |관련 질문
vote |VARCHAR(191) |투표 현황
created_at |DATETIME |생성 일자
updated_at |DATETIME |업데이트 일자
status |LONGTEXT |qa 상태
token_count |BIGINT |사용 토큰 수
llm_provider |VARCHAR(191) |llm provider
llm_model |VARCHAR(191) |llm model
