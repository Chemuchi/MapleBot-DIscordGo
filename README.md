# 메이플 디스코드 봇 (기초: /검색)

넥슨 OPEN API를 이용해 `/검색 <닉네임>` 슬래시 커맨드로 캐릭터 정보를 임베드로 보여주는 Go 디스코드 봇입니다.

## 동작 흐름

1. `/검색 닉네임:홍길동` 실행
2. `GET /maplestory/v1/id?character_name=홍길동` → `ocid` 획득
3. `GET /maplestory/v1/character/basic?ocid={ocid}` → 캐릭터 상세 정보 획득
4. 결과를 디스코드 임베드로 응답 (실패 시 에러 임베드로 원인 안내: 400/404/429 등)

API 응답 대기 중 타임아웃(3초) 문제를 피하기 위해 `InteractionResponseDeferredChannelMessageWithSource`로
먼저 "생각 중..." 응답을 보낸 뒤, 결과를 `InteractionResponseEdit`으로 수정하는 방식을 사용했습니다.

## 준비물

- Go 1.22 이상
- Discord 봇 토큰 (Discord Developer Portal에서 발급, `applications.commands`, `bot` 스코프 필요)
- 넥슨 오픈 API 키 (https://openapi.nexon.com 에서 발급)

## 설치 및 실행

```bash
# 1. 의존성 설치 (인터넷 연결 필요, proxy.golang.org 접근 가능한 일반 환경에서 실행)
go mod tidy

# 2. .env 파일 생성 (.env.example을 복사해서 실제 값 채우기)
cp .env.example .env
# .env 파일을 열어서 DISCORD_BOT_TOKEN, NEXON_API_KEY 값을 채워넣기

# 3. 실행
go run main.go
```

`.env` 파일은 `godotenv` 패키지로 프로그램 시작 시 자동으로 읽어옵니다.
`.gitignore`에 `.env`가 등록되어 있어서 실수로 깃 저장소에 토큰이 올라가는 걸 막아줍니다.
(즉, 절대 `.env`를 직접 `git add`하지 마세요. 팀원과 공유할 땐 `.env.example`만 커밋하세요.)

## 주의사항

- `session.ApplicationCommandCreate`에서 두 번째 인자(guildID)를 빈 문자열로 두면
  **글로벌 커맨드**로 등록됩니다. 전 서버에 반영되는 데 최대 1시간 정도 걸릴 수 있어요.
  개발 중에는 아래처럼 특정 서버 ID를 넣으면 즉시 반영됩니다.

  ```go
  session.ApplicationCommandCreate(session.State.User.ID, "여기에_길드ID", cmd)
  ```

- 넥슨 오픈 API는 요청 한도(rate limit)가 있습니다. `429` 응답이 오면
  "요청 한도를 초과했습니다" 메시지로 안내하도록 이미 처리해두었습니다.
- `character_image` URL은 시간이 지나면 만료/변경될 수 있는 서명된 URL입니다
  (실제 넥슨 API 정책 확인 권장).

## 다음에 이어서 확장하면 좋은 기능

- OCID를 캐시(Redis 등)에 저장해서 동일 닉네임 재검색 시 1차 API 호출 스킵
- `/전투력`, `/장비` 등 다른 캐릭터 API 엔드포인트 추가 (예: `/character/stat`, `/character/item-equipment`)
- 캐릭터 이미지 조회 시 `date` 쿼리 파라미터를 붙여 특정 날짜 스냅샷 조회
- 요청 실패 시 재시도(backoff) 로직 추가
