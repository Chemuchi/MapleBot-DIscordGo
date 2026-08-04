package commands

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"maple-discord-bot/internal/database"
)

const (
	resetConfirmButtonID = "reset:confirm"
	resetCancelButtonID  = "reset:cancel"
)

// ResetCommand는 확인을 받은 뒤 호출한 사용자의 저장 정보를 삭제합니다.
type ResetCommand struct {
	APIKeys *database.UserAPIKeyStore
}

func (c *ResetCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "초기화",
		Description: "저장된 사용자의 API Key를 삭제합니다.",
	}
}

func (c *ResetCommand) ComponentCustomIDs() []string {
	return []string{resetConfirmButtonID, resetCancelButtonID}
}

func (c *ResetCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if c.APIKeys == nil {
		respondEphemeral(s, i, "현재 데이터베이스에 연결할 수 없어 사용자 정보를 초기화할 수 없습니다.")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ DB에 저장된 내 넥슨 API 키와 사용자 정보를 삭제할까요? 삭제 후에는 복구할 수 없습니다.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						CustomID: resetConfirmButtonID,
						Label:    "네.",
						Style:    discordgo.DangerButton,
					},
					discordgo.Button{
						CustomID: resetCancelButtonID,
						Label:    "아니요",
						Style:    discordgo.SecondaryButton,
					},
				}},
			},
		},
	}); err != nil {
		log.Printf("초기화 확인 메시지 표시 실패: %v", err)
	}
}

func (c *ResetCommand) HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.MessageComponentData().CustomID == resetCancelButtonID {
		updateComponentMessage(s, i, "초기화를 취소했습니다.")
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.Printf("초기화 확인 응답 지연 처리 실패: %v", err)
		return
	}

	userID := interactionUserID(i)
	message := "디스코드 사용자 정보를 확인할 수 없어 초기화하지 못했습니다."
	if userID != "" && c.APIKeys != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		deleted, err := c.APIKeys.Delete(ctx, userID)
		cancel()
		switch {
		case err != nil:
			log.Printf("사용자 정보 초기화 실패 (discord_user_id=%s): %v", userID, err)
			message = "사용자 정보 삭제에 실패했습니다. 잠시 후 다시 시도해 주세요."
		case deleted:
			message = "✅ DB에 저장된 내 사용자 정보와 넥슨 API 키를 삭제했습니다."
		default:
			message = "삭제할 사용자 정보가 없습니다. 이미 초기화된 상태입니다."
		}
	}

	emptyComponents := []discordgo.MessageComponent{}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &message,
		Components: &emptyComponents,
	}); err != nil {
		log.Printf("초기화 결과 메시지 수정 실패: %v", err)
	}
}

func updateComponentMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: []discordgo.MessageComponent{},
		},
	}); err != nil {
		log.Printf("초기화 확인 메시지 수정 실패: %v", err)
	}
}
