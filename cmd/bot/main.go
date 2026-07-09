// 메이플 디스코드 봇의 진입점.
// 여기서는 "설정 로드 -> 넥슨 클라이언트 생성 -> 커맨드 등록 -> 봇 실행"의
// 흐름만 다루고, 실제 로직은 각 internal 패키지에 있습니다.
//
// 새 커맨드를 추가하려면:
//  1. internal/nexon 에 필요한 API 호출 함수 추가 (예: GetCharacterStat)
//  2. internal/discord/commands 에 새 파일 추가 (예: stat.go), discord.Command 구현
//  3. 아래 bot.Register(...) 줄에 한 줄 추가
package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"maple-discord-bot/internal/config"
	"maple-discord-bot/internal/database"
	"maple-discord-bot/internal/discord"
	"maple-discord-bot/internal/discord/commands"
	"maple-discord-bot/internal/nexon"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}

	nexonClient := nexon.NewClient(cfg.NexonAPIKey)

	// 데이터베이스 초기화
	var db *sql.DB
	if cfg.DatabaseURL != "" {
		var dbErr error
		db, dbErr = database.Connect(cfg.DatabaseURL)
		if dbErr != nil {
			log.Printf("데이터베이스 초기 연결 실패 (봇 구동은 계속 진행): %v", dbErr)
		} else {
			log.Println("데이터베이스 연결 성공!")
			defer db.Close()
		}
	} else {
		log.Println("DATABASE_URL이 설정되지 않아 데이터베이스 연동 없이 봇을 시작합니다.")
	}

	bot, err := discord.New(cfg.DiscordBotToken, nexonClient, cfg.SundayChannelID, db)
	if err != nil {
		log.Fatalf("봇 생성 실패: %v", err)
	}

	// ---- 커맨드 등록 ----
	// 새 기능을 추가할 때마다 여기에 한 줄씩 추가하면 됩니다.
	bot.Register(&commands.SearchCommand{Nexon: nexonClient})
	bot.Register(&commands.ScheduleCommand{Nexon: nexonClient})
	bot.Register(&commands.BotInfoCommand{})
	bot.Register(&commands.BotStatusCommand{DB: db})



	if err := bot.Run(); err != nil {
		log.Fatalf("봇 실행 실패: %v", err)
	}
	defer bot.Close()

	log.Println("모든 준비 완료. Ctrl+C로 종료하세요.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("종료 중...")
}
