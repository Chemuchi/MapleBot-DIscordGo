// Package commands는 각 슬래시 커맨드의 실제 구현을 모아둡니다.
// 새 커맨드를 추가할 땐 이 디렉토리에 파일을 하나 새로 만들고,
// discord.Command 인터페이스(Definition, Handle)를 구현하면 됩니다.
package commands

import (
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"maple-discord-bot/internal/format"
	"maple-discord-bot/internal/nexon"
)

const (
	embedColorSuccess = 0x00A99D // 메이플 로고 계열 청록색
	embedColorError   = 0xE74C3C // 에러용 빨간색
)

// SearchCommand : "/검색 <닉네임>" 커맨드.
// 넥슨 API 호출은 nexon.Client에 위임하고, 이 파일은
// "디스코드 커맨드 정의 + 결과를 임베드로 바꾸는 것"만 책임집니다.
type SearchCommand struct {
	Nexon *nexon.Client
}

// Definition : 디스코드에 등록할 슬래시 커맨드 정의
func (c *SearchCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
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
	}
}

// Handle : "/검색" 실행 시 호출되는 핸들러
func (c *SearchCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// API 호출에 시간이 걸릴 수 있으므로 우선 "생각 중" 응답으로 defer 처리
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("interaction defer 실패: %v", err)
		return
	}

	characterName := i.ApplicationCommandData().Options[0].StringValue()

	character, err := c.Nexon.SearchCharacterByName(characterName)

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

func buildCharacterEmbed(ch *nexon.CharacterBasic) *discordgo.MessageEmbed {
	guildName := ch.CharacterGuildName
	if guildName == "" {
		guildName = "無소속"
	}

	return &discordgo.MessageEmbed{
		Title: ch.CharacterName + " 님의 캐릭터 정보",
		Color: embedColorSuccess,
		Image: &discordgo.MessageEmbedImage{
			URL: ch.CharacterImage,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "월드", Value: ch.WorldName, Inline: true},
			{Name: "직업", Value: ch.CharacterClass, Inline: true},
			{Name: "레벨", Value: itoa(ch.CharacterLevel), Inline: true},
			{Name: "길드", Value: guildName, Inline: true},
			{Name: "경험치", Value: ch.CharacterExpRate + "%", Inline: true},
			{Name: "캐릭터 생성일", Value: format.Date(ch.CharacterDateCreate), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Powered by NEXON Open API",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func buildErrorEmbed(characterName string, err error) *discordgo.MessageEmbed {
	message := err.Error()

	if apiErr, ok := err.(*nexon.APIError); ok {
		message = apiErr.FriendlyMessage(characterName)
	}

	return &discordgo.MessageEmbed{
		Title:       "❌ 검색 실패",
		Description: message,
		Color:       embedColorError,
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
