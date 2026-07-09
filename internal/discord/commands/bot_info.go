package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

// 만약 기존 search.go와 동일한 패키지(commands) 내에 선언된다면
// embedColorSuccess를 그대로 공유하여 사용할 수 있습니다.
// 여기서는 다른 파일과의 독립성을 위해 로컬 상수로 재정의하거나 기존 값을 참조합니다.
const (
	botInfoColor = 0x00A99D // 메이플 로고 계열 청록색
)

type BotInfoCommand struct{}

// Definition : 디스코드에 등록할 /봇정보 명령어 스펙을 정의합니다.
func (c *BotInfoCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "봇정보",
		Description: "현재 실행 중인 봇의 배포 버전 정보를 확인합니다.",
	}
}

// Handle : /봇정보 명령어가 입력되었을 때 주입된 환경변수를 파싱하여 응답합니다.
func (c *BotInfoCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. 환경 변수 읽기 및 로컬 개발 환경(내 컴퓨터)을 위한 예외 처리
	commitSHA := os.Getenv("COMMIT_SHA")
	if commitSHA == "" {
		commitSHA = "development (local)"
	}

	// 2. 기존 캐릭터 검색 포맷과 통일성을 맞춘 임베드 생성
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 MapleBot 배포 정보",
		Description: "현재 컨테이너 인프라에서 실행 중인 봇의 배포 메타데이터입니다.",
		Color:       botInfoColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Git 커밋 해시", Value: fmt.Sprintf("`%s`", commitSHA), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 3. 외부 통신이나 무거운 연산이 없으므로 Defer 처리 없이 즉시 채널에 전송
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		fmt.Printf("봇정보 명령어 응답 실패: %v\n", err)
	}
}
