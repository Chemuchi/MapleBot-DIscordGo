package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// ==================== 설정 ====================

const (
	nexonAPIBase   = "https://open.api.nexon.com"
	requestTimeout = 10 * time.Second
)

var (
	discordToken string
	nexonAPIKey  string
	httpClient   = &http.Client{Timeout: requestTimeout}
)

// ==================== 넥슨 API 응답 구조체 ====================

// OcidResponse : 닉네임 -> OCID 조회 응답
type OcidResponse struct {
	Ocid string `json:"ocid"`
}

// NexonErrorResponse : 400/404/429 등 에러 응답 공통 포맷
type NexonErrorResponse struct {
	Error struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error"`
}

// CharacterBasic : 캐릭터 기본 정보 응답
type CharacterBasic struct {
	Date                 *string `json:"date"`
	CharacterName        string  `json:"character_name"`
	WorldName            string  `json:"world_name"`
	CharacterGender      string  `json:"character_gender"`
	CharacterClass       string  `json:"character_class"`
	CharacterClassLevel  string  `json:"character_class_level"`
	CharacterLevel       int     `json:"character_level"`
	CharacterExp         int64   `json:"character_exp"`
	CharacterExpRate     string  `json:"character_exp_rate"`
	CharacterGuildName   string  `json:"character_guild_name"`
	CharacterImage       string  `json:"character_image"`
	CharacterDateCreate  string  `json:"character_date_create"`
	AccessFlag           string  `json:"access_flag"`
	LiberationQuestClear string  `json:"liberation_quest_clear"`
}

// nexonAPIError : 넥슨 API 호출 실패 시 사용하는 커스텀 에러 타입
// (사용자에게 그대로 보여줄 수 있도록 메시지를 담아둠)
type nexonAPIError struct {
	StatusCode int
	Name       string
	Message    string
}

func (e *nexonAPIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.StatusCode, e.Name, e.Message)
}

// ==================== 넥슨 API 호출 함수 ====================

// callNexonAPI : 넥슨 오픈 API에 GET 요청을 보내고 바디를 반환하는 공통 함수
func callNexonAPI(endpoint string, query url.Values) ([]byte, int, error) {
	reqURL := nexonAPIBase + endpoint
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("요청 생성 실패: %w", err)
	}
	req.Header.Set("x-nxopen-api-key", nexonAPIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("요청 전송 실패: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("응답 읽기 실패: %w", err)
	}

	return body, resp.StatusCode, nil
}

// parseNexonError : 에러 응답 바디를 파싱해서 nexonAPIError로 변환
func parseNexonError(statusCode int, body []byte) error {
	var errResp NexonErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// 에러 바디 파싱조차 실패한 경우, 상태 코드만이라도 알려줌
		return &nexonAPIError{
			StatusCode: statusCode,
			Name:       "UNKNOWN_ERROR",
			Message:    fmt.Sprintf("알 수 없는 오류가 발생했습니다 (status: %d)", statusCode),
		}
	}
	return &nexonAPIError{
		StatusCode: statusCode,
		Name:       errResp.Error.Name,
		Message:    errResp.Error.Message,
	}
}

// getOcid : 캐릭터 닉네임으로 OCID 조회 (1차 API 호출)
func getOcid(characterName string) (string, error) {
	query := url.Values{}
	query.Set("character_name", characterName)

	body, statusCode, err := callNexonAPI("/maplestory/v1/id", query)
	if err != nil {
		return "", err
	}

	if statusCode != http.StatusOK {
		return "", parseNexonError(statusCode, body)
	}

	var ocidResp OcidResponse
	if err := json.Unmarshal(body, &ocidResp); err != nil {
		return "", fmt.Errorf("OCID 응답 파싱 실패: %w", err)
	}

	return ocidResp.Ocid, nil
}

// getCharacterBasic : OCID로 캐릭터 기본 정보 조회 (2차 API 호출)
func getCharacterBasic(ocid string) (*CharacterBasic, error) {
	query := url.Values{}
	query.Set("ocid", ocid)

	body, statusCode, err := callNexonAPI("/maplestory/v1/character/basic", query)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, parseNexonError(statusCode, body)
	}

	var basic CharacterBasic
	if err := json.Unmarshal(body, &basic); err != nil {
		return nil, fmt.Errorf("캐릭터 정보 응답 파싱 실패: %w", err)
	}

	return &basic, nil
}

// searchCharacter : 닉네임 하나로 OCID 조회 -> 캐릭터 정보 조회까지 한번에 처리
func searchCharacter(characterName string) (*CharacterBasic, error) {
	ocid, err := getOcid(characterName)
	if err != nil {
		return nil, err
	}

	basic, err := getCharacterBasic(ocid)
	if err != nil {
		return nil, err
	}

	return basic, nil
}

// ==================== 임베드 빌드 ====================

// jobColor : 직업군 등에 상관없이 통일감 있는 메이플 느낌의 색상 사용
const embedColorSuccess = 0x00A99D // 메이플 로고 계열 청록색
const embedColorError = 0xE74C3C   // 에러용 빨간색

func buildCharacterEmbed(c *CharacterBasic) *discordgo.MessageEmbed {
	guildName := c.CharacterGuildName
	if guildName == "" {
		guildName = "無소속"
	}

	// 경험치는 콤마 포맷으로 가독성 개선
	expFormatted := formatWithComma(c.CharacterExp)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s 님의 캐릭터 정보", c.CharacterName),
		Color: embedColorSuccess,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: c.CharacterImage,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "월드", Value: c.WorldName, Inline: true},
			{Name: "직업", Value: fmt.Sprintf("%s (%s차)", c.CharacterClass, c.CharacterClassLevel), Inline: true},
			{Name: "레벨", Value: fmt.Sprintf("%d", c.CharacterLevel), Inline: true},
			{Name: "길드", Value: guildName, Inline: true},
			{Name: "경험치", Value: expFormatted, Inline: true},
			{Name: "경험치율", Value: fmt.Sprintf("%s%%", c.CharacterExpRate), Inline: true},
			{Name: "캐릭터 생성일", Value: formatDate(c.CharacterDateCreate), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by NEXON Open API",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed
}

func buildErrorEmbed(characterName string, err error) *discordgo.MessageEmbed {
	message := err.Error()

	// nexonAPIError인 경우 사용자 친화적인 메시지로 가공
	if apiErr, ok := err.(*nexonAPIError); ok {
		switch apiErr.StatusCode {
		case http.StatusBadRequest:
			message = "요청 형식이 올바르지 않습니다."
		case http.StatusNotFound:
			message = fmt.Sprintf("'%s' 캐릭터를 찾을 수 없습니다. 닉네임을 다시 확인해주세요.", characterName)
		case http.StatusTooManyRequests:
			message = "API 요청 한도를 초과했습니다. 잠시 후 다시 시도해주세요."
		default:
			message = fmt.Sprintf("%s (%s)", apiErr.Message, apiErr.Name)
		}
	}

	return &discordgo.MessageEmbed{
		Title:       "❌ 검색 실패",
		Description: message,
		Color:       embedColorError,
	}
}

// ==================== 유틸 ====================

func formatWithComma(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var result []byte
	for i, c := range []byte(s) {
		if i != 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	if neg {
		return "-" + string(result)
	}
	return string(result)
}

func formatDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format("2006년 01월 02일")
}

// ==================== 디스코드 커맨드 ====================

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "검색",
		Description: "메이플스토리 캐릭터 정보를 검색합니다.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "닉네임",
				Description: "검색할 캐릭터 닉네임",
				Required:    true,
			},
		},
	},
}

func handleSearchCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// API 호출에 시간이 걸릴 수 있으므로 우선 "생각 중" 응답으로 defer 처리
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("interaction defer 실패: %v", err)
		return
	}

	options := i.ApplicationCommandData().Options
	characterName := options[0].StringValue()

	character, err := searchCharacter(characterName)

	var embed *discordgo.MessageEmbed
	if err != nil {
		log.Printf("캐릭터 검색 실패 (%s): %v", characterName, err)
		embed = buildErrorEmbed(characterName, err)
	} else {
		embed = buildCharacterEmbed(character)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		log.Printf("interaction 응답 수정 실패: %v", err)
	}
}

func interactionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "검색":
		handleSearchCommand(s, i)
	}
}

// ==================== main ====================

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("환경변수 %s 가 설정되지 않았습니다.", key)
	}
	return v
}

func main() {
	// .env 파일이 있으면 읽어서 환경변수로 등록 (없어도 에러로 취급하지 않음:
	// 배포 환경에서는 .env 없이 시스템 환경변수로 직접 넣는 경우도 많기 때문)
	if err := godotenv.Load(); err != nil {
		log.Println(".env 파일을 찾을 수 없습니다. 시스템 환경변수를 사용합니다.")
	}

	discordToken = mustGetEnv("DISCORD_BOT_TOKEN")
	nexonAPIKey = mustGetEnv("NEXON_API_KEY")

	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("디스코드 세션 생성 실패: %v", err)
	}

	session.AddHandler(interactionCreateHandler)
	session.Identify.Intents = discordgo.IntentsNone

	if err := session.Open(); err != nil {
		log.Fatalf("디스코드 연결 실패: %v", err)
	}
	defer session.Close()

	log.Println("봇이 시작되었습니다. 슬래시 커맨드를 등록합니다...")

	// 길드 한정이 아닌 글로벌 커맨드로 등록 (전파에 최대 1시간 소요될 수 있음)
	// 테스트 시엔 특정 길드 ID를 넣어 즉시 반영되는 방식을 권장합니다.
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for idx, cmd := range commands {
		created, err := session.ApplicationCommandCreate(session.State.User.ID, "", cmd)
		if err != nil {
			log.Fatalf("커맨드 등록 실패 (%s): %v", cmd.Name, err)
		}
		registeredCommands[idx] = created
	}

	log.Println("모든 커맨드 등록 완료. Ctrl+C로 종료하세요.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("종료 중...")
}
