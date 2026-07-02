package commands

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"maple-discord-bot/internal/nexon"
	"net/http"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disintegration/imaging"
)

const (
	embedColorSuccess = 0x00A99D // 메이플 로고 계열 청록색
	embedColorError   = 0xE74C3C // 에러용 빨간색
)

type SearchCommand struct {
	Nexon *nexon.Client
}

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

func (c *SearchCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("interaction defer 실패: %v", err)
		return
	}

	characterName := i.ApplicationCommandData().Options[0].StringValue()
	character, err := c.Nexon.SearchCharacterByName(characterName)

	if err != nil {
		log.Printf("캐릭터 검색 실패 (%s): %v", characterName, err)
		embed := buildErrorEmbed(characterName, err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		return
	}

	// 1. 이미지 크롭 처리 (메모리 버퍼로 받아옴)
	imageBuf, err := cropCharacterImage(character.CharacterImage)
	var embed *discordgo.MessageEmbed
	var files []*discordgo.File

	if err != nil {
		log.Printf("이미지 크롭 실패 (기본 이미지로 대체): %v", err)
		// 크롭 실패 시 에러는 안 띄우고 원본 URL로 빌드
		embed = buildCharacterEmbed(character, character.CharacterImage, false)
	} else {
		// 크롭 성공 시 attachment 포맷 적용 및 파일 배열 준비
		embed = buildCharacterEmbed(character, "attachment://character.png", true)
		files = []*discordgo.File{
			{
				Name:        "character.png",
				ContentType: "image/png",
				Reader:      imageBuf,
			},
		}
	}

	// 2. 파일과 임베드를 동시에 Interaction 응답으로 전송
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files:  files,
	})
	if err != nil {
		log.Printf("interaction 응답 수정 실패: %v", err)
	}
}

// cropCharacterImage : 넥슨 이미지 URL을 받아 중심부를 크롭한 뒤 PNG 바이너리 버퍼를 리턴합니다.
func cropCharacterImage(imageURL string) (*bytes.Buffer, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	srcImg, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	// 크롭사이즈

	bounds := srcImg.Bounds()
	cropWidth := int(float64(bounds.Dx()) * 0.35)  // 원본 가로의 55% 크기로
	cropHeight := int(float64(bounds.Dy()) * 0.35) // 원본 세로의 55% 크기로

	// 중심(Center)을 기준으로 크롭
	croppedImg := imaging.CropAnchor(srcImg, cropWidth, cropHeight, imaging.Center)

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, croppedImg); err != nil {
		return nil, err
	}

	return buf, nil
}

// buildCharacterEmbed : 이제 Image가 아닌 Thumbnail을 사용하도록 변경되었습니다.
func buildCharacterEmbed(ch *nexon.CharacterBasic, imagePath string, isAttachment bool) *discordgo.MessageEmbed {
	guildName := ch.CharacterGuildName
	if guildName == "" {
		guildName = "無소속"
	}

	return &discordgo.MessageEmbed{
		Title: ch.CharacterName + " 님의 캐릭터 정보",
		Color: embedColorSuccess,
		// 기존 Image 대신 Thumbnail 포맷으로 변경
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: imagePath,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "월드", Value: ch.WorldName, Inline: true},
			{Name: "직업", Value: ch.CharacterClass, Inline: true},
			{Name: "레벨", Value: itoa(ch.CharacterLevel), Inline: true},
			{Name: "길드", Value: guildName, Inline: true},
			{Name: "경험치", Value: ch.CharacterExpRate + "%", Inline: true},
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
