// Package config는 .env 파일 및 시스템 환경변수를 읽어
// 봇 실행에 필요한 설정값을 제공합니다.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config : 봇 구동에 필요한 전체 설정
type Config struct {
	DiscordBotToken string
	NexonAPIKey     string
	SundayChannelID string // 썬데이 메이플 공지 전송 대상 채널 ID
	DatabaseURL     string // Supabase 데이터베이스 연결 URL
}

// Load : .env 파일(있으면)을 읽어들인 뒤, 필수 환경변수를 검증하여 Config를 반환합니다.
// .env 파일이 없어도 에러로 취급하지 않습니다 (배포 환경에서는
// 시스템 환경변수로 직접 주입하는 경우가 많기 때문).
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println(".env 파일을 찾을 수 없습니다. 시스템 환경변수를 사용합니다.")
	}

	token, err := mustGetEnv("DISCORD_BOT_TOKEN")
	if err != nil {
		return nil, err
	}

	apiKey, err := mustGetEnv("NEXON_API_KEY")
	if err != nil {
		return nil, err
	}

	// 썬데이 채널 ID는 선택적으로 로드 (설정 안 되어있으면 스케줄러는 작동 안함)
	sundayChannelID := os.Getenv("SUNDAY_CHANNEL_ID")

	// 데이터베이스 URL 역시 선택적으로 로드
	databaseURL := os.Getenv("DATABASE_URL")

	return &Config{
		DiscordBotToken: token,
		NexonAPIKey:     apiKey,
		SundayChannelID: sundayChannelID,
		DatabaseURL:     databaseURL,
	}, nil
}

func mustGetEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("환경변수 %s 가 설정되지 않았습니다", key)
	}
	return v, nil
}
