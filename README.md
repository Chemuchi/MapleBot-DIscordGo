# 메이플 디스코드 봇

넥슨 OPEN API를 이용해 슬래시 커맨드로 메이플스토리 캐릭터 정보를 조회하는 Go 디스코드 봇입니다.

## 프로젝트 구조

```
maple-discord-bot/
├── go.mod
├── .env.example
├── .gitignore
├── cmd/
│   └── bot/
│       └── main.go              # 진입점: 설정 로드 -> 클라이언트/봇 생성 -> 실행
└── internal/
    ├── config/
    │   └── config.go            # .env / 환경변수 로딩
    ├── nexon/                    # 넥슨 API 레이어 (디스코드를 전혀 모름)
    │   ├── client.go             # 공통 HTTP 호출, 인증 헤더
    │   ├── errors.go             # APIError 타입, 상태 코드별 에러 메시지
    │   └── character.go          # GetOcid, GetCharacterBasic, 관련 타입
    ├── format/
    │   └── format.go             # 콤마 포맷, 날짜 포맷 등 순수 유틸
    └── discord/
        ├── command.go            # 모든 커맨드가 구현할 Command 인터페이스
        ├── bot.go                # 세션 생성, 커맨드 등록/디스패치
        └── commands/
            └── search.go         # /검색 커맨드 구현
```

**설계 원칙**: `nexon` 패키지는 "메이플 API 클라이언트"고, `discord` 패키지는 "봇 인프라"고,
`discord/commands`는 "커맨드 하나하나"입니다. 서로 책임이 겹치지 않게 나눠서, 기능이
늘어나도 기존 파일을 거의 건드리지 않고 새 파일만 추가하면 되도록 했습니다.

## 새 커맨드 추가하는 법 (예: `/전투력`)

1. **`internal/nexon/`**에 필요한 API 호출 함수 추가
   (예: `character.go`에 `GetCharacterStat(ocid string) (*CharacterStat, error)` 추가,
   또는 새 파일 `stat.go`로 분리)
2. **`internal/discord/commands/`**에 새 파일 생성 (예: `stat.go`)
   ```go
   type StatCommand struct {
       Nexon *nexon.Client
   }
   func (c *StatCommand) Definition() *discordgo.ApplicationCommand { ... }
   func (c *StatCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) { ... }
   ```
   (`search.go`를 그대로 복사해서 이름만 바꾸면 뼈대는 끝)
3. **`cmd/bot/main.go`**의 등록 부분에 한 줄 추가
   ```go
   bot.Register(&commands.StatCommand{Nexon: nexonClient})
   ```

기존 `search.go`, `bot.go`, `client.go`는 전혀 건드릴 필요가 없습니다.

## 동작 흐름 (/검색)

1. `/검색 닉네임:홍길동` 실행
2. `nexon.Client.GetOcid()` → `GET /maplestory/v1/id?character_name=홍길동` → `ocid` 획득
3. `nexon.Client.GetCharacterBasic()` → `GET /maplestory/v1/character/basic?ocid={ocid}` → 캐릭터 상세 정보 획득
4. `discord/commands/search.go`가 결과를 임베드로 변환해 응답 (실패 시 400/404/429별 에러 메시지)

디스코드 3초 응답 제한을 피하기 위해 `InteractionResponseDeferredChannelMessageWithSource`로
먼저 "생각 중..." 응답을 보낸 뒤, 결과를 `InteractionResponseEdit`으로 수정하는 방식을 사용했습니다.

## 준비물

- Go 1.22 이상
- Discord 봇 토큰 (Discord Developer Portal에서 발급, `applications.commands`, `bot` 스코프 필요)
- 넥슨 오픈 API 키 (https://openapi.nexon.com 에서 발급)

## 설치 및 실행

```bash
# 1. 의존성 설치
go mod tidy

# 2. .env 파일 생성 (.env.example을 복사해서 실제 값 채우기)
cp .env.example .env
# .env 파일을 열어서 토큰과 API 키 값을 채워넣기
# 사용자 API Key 암호화 키 생성 (출력값을 API_KEY_ENCRYPTION_KEY에 입력)
openssl rand -base64 32

# 3. 실행 (진입점이 cmd/bot/main.go로 이동했습니다)
go run ./cmd/bot
```

### Docker Compose로 로컬 MySQL과 실행

`.env.example`을 `.env`로 복사한 뒤 Discord/Nexon 값을 채우고 실행합니다.
로컬에서 명령어를 즉시 반영하려면 개발용 디스코드 서버 ID도 설정합니다.

```env
DISCORD_GUILD_ID="개발용 서버 ID"
```

```bash
docker compose up --build -d
docker compose ps
docker compose logs -f bot
```

봇은 Compose 네트워크에서 `mysql:3306`에 연결하며, MySQL health check가 통과한 뒤 시작합니다.
데이터는 `mysql_data` named volume에 보존됩니다. `docker compose down -v`는 데이터를 삭제하므로 사용하지 마세요.
봇 시작 시 `user_api_keys` 테이블이 자동 생성되며 `/세팅`으로 입력받은 API Key는
AES-256-GCM 암호문으로만 저장됩니다. 같은 API Key도 매번 무작위 nonce를 사용하므로 서로 다른 암호문이 됩니다.

### API Key 암호화 키 교체

암호화 키는 일반적인 "시드"와 달리 잃어버리면 기존 데이터를 복구할 수 없는 비밀 키입니다.
키를 바꿀 때는 기존 키를 즉시 삭제하지 말고 아래 순서로 교체합니다.

1. 기존 `API_KEY_ENCRYPTION_KEY` 값을 `API_KEY_ENCRYPTION_PREVIOUS_KEYS`에 옮깁니다.
2. `openssl rand -base64 32`로 새 키를 만들고 `API_KEY_ENCRYPTION_KEY`에 설정합니다.
3. 봇을 재시작합니다. 시작 과정에서 기존 데이터 전체가 이전 키로 복호화된 뒤 새 키로 자동 재암호화됩니다.
4. 로그에서 `사용자 API 키 암호화 저장소 초기화 성공!`을 확인한 뒤 이전 키 환경변수를 제거합니다.

이전 키가 여러 개라면 `API_KEY_ENCRYPTION_PREVIOUS_KEYS="이전키1,이전키2"`처럼 쉼표로 구분합니다.
암호화 키는 `.env`, 배포 환경의 Secret Manager 등에서만 관리하고 Git에는 커밋하지 마세요.

`.env` 파일은 `godotenv` 패키지로 프로그램 시작 시 자동으로 읽어옵니다.
`.gitignore`에 `.env`가 등록되어 있어서 실수로 깃 저장소에 토큰이 올라가는 걸 막아줍니다.
(즉, 절대 `.env`를 직접 `git add`하지 마세요. 팀원과 공유할 땐 `.env.example`만 커밋하세요.)

## 주의사항

- 로컬 `.env`에 `DISCORD_GUILD_ID`가 있으면 해당 서버에 길드 커맨드로 등록되어 바로 반영됩니다.
  배포 환경처럼 값이 없으면 글로벌 커맨드로 등록되며 전 서버 반영까지 최대 1시간 정도 걸릴 수 있습니다.
- 넥슨 오픈 API는 요청 한도(rate limit)가 있습니다. `429` 응답은
  "요청 한도를 초과했습니다" 메시지로 안내하도록 이미 처리해두었습니다.
- `character_image` URL은 시간이 지나면 만료/변경될 수 있는 서명된 URL입니다.
- **썬데이 메이플 자동 공지**: 매주 금요일 오전 10시 5분(KST)에 백그라운드 스케줄러가 돌아가며 최신 썬데이 메이플 공지를 조회하여 전송합니다. 동작을 위해서는 `.env`에 `SUNDAY_CHANNEL_ID`가 올바르게 기입되어 있어야 합니다.


## 다음에 이어서 확장하면 좋은 기능

- OCID를 캐시(Redis 등)에 저장해서 동일 닉네임 재검색 시 1차 API 호출 스킵
  (`internal/nexon`에 캐시 레이어를 끼워 넣으면 `discord/commands`는 그대로 둬도 됨)
- `/전투력`, `/장비` 등 다른 캐릭터 API 엔드포인트 추가 (예: `/character/stat`, `/character/item-equipment`)
- 요청 실패 시 재시도(backoff) 로직을 `nexon/client.go`에 추가
- `internal/discord/commands` 패키지에 단위 테스트 추가 (임베드 빌드 로직은 순수 함수라 테스트하기 쉬움)
