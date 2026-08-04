package discord

import "github.com/bwmarrin/discordgo"

// Command : 슬래시 커맨드 하나가 구현해야 하는 인터페이스.
// 새 기능(/전투력, /장비 등)을 추가할 땐 이 인터페이스를 구현하는
// 파일을 internal/discord/commands/ 아래에 새로 만들고, cmd/bot/main.go에서
// bot.Register(...)에 한 줄만 추가하면 됩니다.
type Command interface {
	// Definition : 디스코드에 등록할 슬래시 커맨드 정의 (이름, 설명, 옵션)
	Definition() *discordgo.ApplicationCommand

	// Handle : 실제 사용자가 커맨드를 실행했을 때 호출되는 핸들러
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// ModalCommand : 슬래시 커맨드에서 연 모달 제출까지 처리하는 커맨드가 구현합니다.
type ModalCommand interface {
	Command

	ModalCustomID() string
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// ComponentCommand : 버튼이나 셀렉트 메뉴 인터랙션까지 처리하는 커맨드가 구현합니다.
type ComponentCommand interface {
	Command

	ComponentCustomIDs() []string
	HandleComponent(s *discordgo.Session, i *discordgo.InteractionCreate)
}
