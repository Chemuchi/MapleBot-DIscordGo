package commands

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"maple-discord-bot/internal/database"
)

const (
	settingsModalID = "settings:nexon-api-key"
	apiKeyInputID   = "nexon-api-key"
)

// SettingsCommand는 사용자별 넥슨 Open API 키를 등록하거나 갱신합니다.
type SettingsCommand struct {
	APIKeys *database.UserAPIKeyStore
}

func (c *SettingsCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "세팅",
		Description: "개인 넥슨 Open API 키를 등록하거나 변경합니다.",
	}
}

func (c *SettingsCommand) ModalCustomID() string {
	return settingsModalID
}

func (c *SettingsCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.APIKeys == nil {
		respondEphemeral(s, i, "현재 데이터베이스에 연결할 수 없어 API 키를 등록할 수 없습니다. 잠시 후 다시 시도해 주세요.")
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: settingsModalID,
			Title:    "넥슨 Open API 키 설정",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    apiKeyInputID,
						Label:       "정보는 암호화 되어 DB에 저장됩니다.",
						Style:       discordgo.TextInputShort,
						Placeholder: "Nexon Open API Key",
						Required:    true,
						MinLength:   1,
						MaxLength:   256,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("세팅 모달 표시 실패: %v", err)
	}
}

func (c *SettingsCommand) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.APIKeys == nil {
		respondEphemeral(s, i, "현재 데이터베이스에 연결할 수 없어 API 키를 등록할 수 없습니다. 잠시 후 다시 시도해 주세요.")
		return
	}

	apiKey := strings.TrimSpace(modalTextValue(i.ModalSubmitData(), apiKeyInputID))
	if apiKey == "" {
		respondEphemeral(s, i, "API 키를 입력해 주세요.")
		return
	}

	userID := interactionUserID(i)
	if userID == "" {
		respondEphemeral(s, i, "디스코드 사용자 정보를 확인할 수 없습니다. 다시 시도해 주세요.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.APIKeys.Save(ctx, userID, apiKey); err != nil {
		log.Printf("사용자 API 키 저장 실패 (discord_user_id=%s): %v", userID, err)
		respondEphemeral(s, i, "API 키 저장에 실패했습니다. 잠시 후 다시 시도해 주세요.")
		return
	}

	respondEphemeral(s, i, "✅ 넥슨 Open API 키를 저장했습니다. 다시 `/세팅`을 실행하면 기존 키를 변경할 수 있습니다.")
}

func modalTextValue(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, component := range data.Components {
		row, ok := component.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, child := range row.Components {
			input, ok := child.(*discordgo.TextInput)
			if ok && input.CustomID == customID {
				return input.Value
			}
		}
	}
	return ""
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("비공개 interaction 응답 실패: %v", err)
	}
}
