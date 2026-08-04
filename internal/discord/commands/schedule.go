package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"maple-discord-bot/internal/database"
	"maple-discord-bot/internal/nexon"
)

type schedulerAPIKeyFinder interface {
	Find(ctx context.Context, discordUserID string) (string, error)
}

// ScheduleCommand : "/스케줄러 <닉네임> [날짜]" 커맨드.
// 캐릭터의 메이플 스케줄러(일일/주간 콘텐츠, 보스 처치) 달성 현황을 보여줍니다.
type ScheduleCommand struct {
	APIKeys *database.UserAPIKeyStore
}

func (c *ScheduleCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "스케줄러",
		Description: "캐릭터의 금일 메이플 스케줄러 상황을 조회합니다.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "닉네임",
				Description: "검색할 캐릭터 닉네임",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "날짜",
				Description: "조회할 날짜 (YYYY-MM-DD, 최대 14일 전까지). 생략하면 오늘 기준 실시간 조회",
				Required:    false,
			},
		},
	}
}

func (c *ScheduleCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := interactionUserID(i)
	if userID == "" {
		respondEphemeral(s, i, "디스코드 사용자 정보를 확인할 수 없습니다. 다시 시도해 주세요.")
		return
	}
	if c.APIKeys == nil {
		respondEphemeral(s, i, "현재 데이터베이스에 연결할 수 없어 스케줄러를 조회할 수 없습니다.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	apiKey, err := lookupSchedulerAPIKey(ctx, c.APIKeys, userID)
	cancel()
	if errors.Is(err, sql.ErrNoRows) || (err == nil && apiKey == "") {
		respondEphemeral(s, i, "등록된 넥슨 Open API 키가 없습니다. 먼저 `/세팅` 명령어로 API 키를 등록해 주세요.")
		return
	}
	if err != nil {
		log.Printf("스케줄러 사용자 API 키 조회 실패 (discord_user_id=%s): %v", userID, err)
		respondEphemeral(s, i, "저장된 API 키를 불러오지 못했습니다. 잠시 후 다시 시도해 주세요.")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("interaction defer 실패: %v", err)
		return
	}

	options := i.ApplicationCommandData().Options
	characterName := options[0].StringValue()

	date := ""
	if len(options) > 1 {
		date = options[1].StringValue()
	}

	userNexonClient := nexon.NewClient(apiKey)
	state, err := userNexonClient.SearchSchedulerByName(characterName, date)

	var embed *discordgo.MessageEmbed
	if err != nil {
		log.Printf("스케줄러 조회 실패 (%s): %v", characterName, err)
		embed = buildErrorEmbed(characterName, err)
	} else {
		embed = buildScheduleEmbed(state)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		log.Printf("interaction 응답 수정 실패: %v", err)
	}
}

func lookupSchedulerAPIKey(ctx context.Context, finder schedulerAPIKeyFinder, userID string) (string, error) {
	return finder.Find(ctx, userID)
}

func buildScheduleEmbed(state *nexon.SchedulerCharacterState) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "주간 보스 처치",
			Value:  fmt.Sprintf("%d / %d", state.WeeklyBossClearCount, state.WeeklyBossClearLimitCount),
			Inline: true,
		},
		{
			Name:   "레벨 / 직업",
			Value:  fmt.Sprintf("%d · %s", state.CharacterLevel, state.CharacterClass),
			Inline: true,
		},
	}

	if bossField := buildBossField(state.BossContents); bossField != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "등록된 보스 처치 현황",
			Value: bossField,
		})
	}

	if dailyField := buildContentField(state.DailyContents); dailyField != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "등록된 일일 컨텐츠",
			Value: dailyField,
		})
	}

	if weeklyField := buildContentField(state.WeeklyContents); weeklyField != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "등록된 주간 컨텐츠",
			Value: weeklyField,
		})
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s 님의 스케줄러 현황 (%s)", state.CharacterName, formatScheduleDate(state.Date)),
		Color:  embedColorSuccess,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by NEXON Open API",
		},
	}
}

// buildBossField : 등록된 보스만 골라서 완료(✅)/미완료(❌)로 표시.
// complete_flag는 명확한 true/false라 신뢰도 높게 처리 가능.
func buildBossField(bosses []nexon.SchedulerBossContent) string {
	var lines []string
	for _, b := range bosses {
		if !b.IsRegistered() {
			continue
		}
		mark := "❌"
		if b.IsComplete() {
			mark = "✅"
		}
		lines = append(lines, fmt.Sprintf("%s (%s) %s", b.ContentName, difficultyKor(b.Difficulty), mark))
	}

	if len(lines) == 0 {
		return "등록된 보스가 없습니다."
	}

	return truncate(strings.Join(lines, "\n"), 1000)
}

// buildContentField : 등록된 일일/주간 콘텐츠·퀘스트 목록을 표시.
// 일일 퀘스트는 quest_state 2를 완료로 보고, 길드 지하수로/플래그레이스는
// now_count를 점수로 표시합니다. 그 외 max_count > 0인 항목은 진행률을 표시합니다.
func buildContentField(contents []nexon.SchedulerContent) string {
	var lines []string
	for _, item := range contents {
		if !item.IsRegistered() {
			continue
		}

		switch {
		case strings.HasPrefix(item.ContentName, "[일일 퀘스트]") || strings.Contains(item.ContentName, "익스트림 몬스터파커"):
			mark := "❌"
			if item.QuestState != nil && *item.QuestState == "2" {
				mark = "✅"
			}
			lines = append(lines, fmt.Sprintf("• %s %s", item.ContentName, mark))
		case strings.HasPrefix(item.ContentName, "에픽 던전"):
			mark := "❌"
			if item.NowCount >= 1 {
				mark = "✅"
			}
			lines = append(lines, fmt.Sprintf("• %s %s", item.ContentName, mark))
		case item.ContentName == "[길드] 지하 수로":
			lines = append(lines, fmt.Sprintf("• [길드] 지하수로: %d점", item.NowCount))
		case item.ContentName == "[길드] 플래그 레이스":
			lines = append(lines, fmt.Sprintf("• [길드] 플래그레이스: %d점", item.NowCount))
		case item.HasProgress():
			lines = append(lines, fmt.Sprintf("• %s (%d/%d)", item.ContentName, item.NowCount, item.MaxCount))
		case item.QuestState != nil:
			lines = append(lines, fmt.Sprintf("• %s (상태값: %s)", item.ContentName, *item.QuestState))
		default:
			lines = append(lines, "• "+item.ContentName)
		}
	}

	if len(lines) == 0 {
		return "등록된 항목이 없습니다."
	}

	return truncate(strings.Join(lines, "\n"), 1000)
}

func difficultyKor(d string) string {
	switch d {
	case "easy":
		return "이지"
	case "normal":
		return "노멀"
	case "hard":
		return "하드"
	case "chaos":
		return "카오스"
	case "extreme":
		return "익스트림"
	default:
		return d
	}
}

// formatScheduleDate : "2026-07-02T00:00+09:00" -> "2026-07-02"
func formatScheduleDate(raw string) string {
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...(생략)"
}
