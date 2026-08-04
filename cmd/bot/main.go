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
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	if cfg.MySQL.Enabled() {
		var dbErr error
		db, dbErr = database.ConnectMySQL(cfg.MySQL)
		if dbErr != nil {
			log.Printf("데이터베이스 초기 연결 실패 (봇 구동은 계속 진행): %v", dbErr)
		} else {
			log.Println("MySQL 데이터베이스 연결 성공!")
			defer db.Close()
		}
	} else if cfg.DatabaseURL != "" {
		// 운영 MySQL 전환 전까지 기존 Supabase 배포를 위한 호환 경로를 유지합니다.
		var dbErr error
		db, dbErr = database.ConnectPostgres(cfg.DatabaseURL)
		if dbErr != nil {
			log.Printf("기존 PostgreSQL 데이터베이스 초기 연결 실패 (봇 구동은 계속 진행): %v", dbErr)
		} else {
			log.Println("기존 PostgreSQL 데이터베이스 연결 성공!")
			defer db.Close()
		}
	} else {
		log.Println("MYSQL_* 또는 DATABASE_URL이 설정되지 않아 데이터베이스 연동 없이 봇을 시작합니다.")
	}

	bot, err := discord.New(cfg.DiscordBotToken, cfg.DiscordGuildID, nexonClient, cfg.SundayChannelID, db)
	if err != nil {
		log.Fatalf("봇 생성 실패: %v", err)
	}

	// ---- 커맨드 등록 ----
	// 새 기능을 추가할 때마다 여기에 한 줄씩 추가하면 됩니다.
	bot.Register(&commands.SearchCommand{Nexon: nexonClient})
	var apiKeyStore *database.UserAPIKeyStore
	if db != nil {
		apiKeyCipher, cipherErr := database.NewAPIKeyCipher(
			cfg.APIKeyEncryptionKey,
			cfg.APIKeyEncryptionPreviousKeys,
		)
		if cipherErr != nil {
			log.Printf("사용자 API 키 암호화 초기화 실패 (/세팅 비활성화): %v", cipherErr)
		} else {
			apiKeyStore = database.NewUserAPIKeyStore(db, apiKeyCipher)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := apiKeyStore.EnsureSchema(ctx); err != nil {
				log.Printf("사용자 API 키 테이블 초기화 실패: %v", err)
				apiKeyStore = nil
			} else {
				log.Println("사용자 API 키 암호화 저장소 초기화 성공!")
			}
			cancel()
		}
	}
	bot.Register(&commands.ScheduleCommand{APIKeys: apiKeyStore})
	bot.Register(&commands.SettingsCommand{APIKeys: apiKeyStore})
	bot.Register(&commands.ResetCommand{APIKeys: apiKeyStore})
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
