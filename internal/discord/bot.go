// Package discord는 디스코드 봇 세션 관리, 슬래시 커맨드 등록/디스패치를 담당합니다.
// 실제 각 커맨드의 비즈니스 로직은 internal/discord/commands 패키지에 있습니다.
package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"maple-discord-bot/internal/nexon"
)

// Bot : 디스코드 세션과 등록된 커맨드들을 관리합니다.
type Bot struct {
	session         *discordgo.Session
	commands        map[string]Command // 커맨드 이름 -> Command 구현체
	nexonClient     *nexon.Client      // 넥슨 API 클라이언트
	sundayChannelID string             // 알림 전송 채널 ID
}

// New : 봇 토큰으로 새 Bot을 생성합니다.
func New(token string, nexonClient *nexon.Client, sundayChannelID string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("디스코드 세션 생성 실패: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsNone

	return &Bot{
		session:         session,
		commands:        make(map[string]Command),
		nexonClient:     nexonClient,
		sundayChannelID: sundayChannelID,
	}, nil
}

// Register : 커맨드를 봇에 등록합니다. main.go에서 사용할 커맨드들을 나열하면 됩니다.
func (b *Bot) Register(cmd Command) {
	name := cmd.Definition().Name
	b.commands[name] = cmd
}

// Run : 디스코드에 연결하고, 슬래시 커맨드를 (글로벌로) 등록한 뒤,
// 인터랙션 핸들러를 붙입니다. 종료 시 Close()를 호출해주세요.
func (b *Bot) Run() error {
	b.session.AddHandler(b.onInteractionCreate)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("디스코드 연결 실패: %w", err)
	}

	log.Println("봇이 연결되었습니다. 슬래시 커맨드를 등록합니다...")
	for name, cmd := range b.commands {
		// 글로벌 커맨드로 등록 (전파에 최대 1시간 소요될 수 있음).
		// 개발 중 즉시 반영이 필요하면 두 번째 인자에 guildID를 넣으세요.
		if _, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd.Definition()); err != nil {
			return fmt.Errorf("커맨드 등록 실패 (%s): %w", name, err)
		}
		log.Printf("커맨드 등록 완료: /%s", name)
	}

	// 썬데이 메이플 백그라운드 알림 루프 실행
	go b.StartSundayNoticeLoop()

	return nil
}

// Close : 디스코드 세션을 종료합니다.
func (b *Bot) Close() error {
	return b.session.Close()
}

// onInteractionCreate : 들어온 인터랙션을 이름에 맞는 Command로 라우팅합니다.
func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	name := i.ApplicationCommandData().Name
	cmd, ok := b.commands[name]
	if !ok {
		log.Printf("등록되지 않은 커맨드 호출됨: %s", name)
		return
	}

	cmd.Handle(s, i)
}
