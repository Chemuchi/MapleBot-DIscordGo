package commands

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type BotStatusCommand struct {
	DB *sql.DB
}

// Definition : 디스코드에 등록할 /봇상태 명령어 스펙을 정의합니다.
func (c *BotStatusCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "봇상태",
		Description: "현재 봇의 실시간 지연 시간(핑) 및 데이터베이스 연결 상태를 확인합니다.",
	}
}

// Handle : /봇상태 명령어가 입력되었을 때 실시간 핑과 실제 DB 상태를 응답합니다.
func (c *BotStatusCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ping := s.HeartbeatLatency().Milliseconds()

	// 데이터베이스 실제 연결 상태 검사
	dbStatus := "🟢 연결됨"
	if c.DB == nil {
		dbStatus = "🔴 미연동 (DATABASE_URL 설정 누락)"
	} else if err := c.DB.Ping(); err != nil {
		dbStatus = fmt.Sprintf("🔴 연결 오류 (%v)", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 MapleBot 실시간 상태",
		Description: "봇의 네트워크 응답 속도 및 외부 리소스 연결 상태입니다.",
		Color:       botInfoColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "디스코드 핑", Value: fmt.Sprintf("`%dms`", ping), Inline: true},
			{Name: "데이터베이스 상태", Value: fmt.Sprintf("`%s`", dbStatus), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("봇상태 명령어 응답 실패: %v\n", err)
	}
}
