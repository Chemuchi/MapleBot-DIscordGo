// Package config는 .env 파일 및 시스템 환경변수를 읽어
// 봇 실행에 필요한 설정값을 제공합니다.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config : 봇 구동에 필요한 전체 설정
type Config struct {
	DiscordBotToken              string
	DiscordGuildID               string // 개발용 길드 ID. 비어 있으면 글로벌 커맨드로 등록
	NexonAPIKey                  string
	SundayChannelID              string // 썬데이 메이플 공지 전송 대상 채널 ID
	MySQL                        MySQLConfig
	DatabaseURL                  string // 운영 전환 전까지 유지하는 Supabase PostgreSQL 연결 URL
	APIKeyEncryptionKey          string
	APIKeyEncryptionPreviousKeys []string
}

// MySQLConfig : 로컬 MySQL 연결에 필요한 개별 설정
type MySQLConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	TLSMode  string
}

// Enabled : MySQL 연결 설정이 하나라도 주입되었는지 반환합니다.
func (c MySQLConfig) Enabled() bool {
	return c.Host != "" || c.Port != "" || c.Database != "" || c.User != "" || c.Password != ""
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
		DiscordGuildID:  os.Getenv("DISCORD_GUILD_ID"),
		NexonAPIKey:     apiKey,
		SundayChannelID: sundayChannelID,
		MySQL: MySQLConfig{
			Host:     os.Getenv("MYSQL_HOST"),
			Port:     os.Getenv("MYSQL_PORT"),
			Database: os.Getenv("MYSQL_DATABASE"),
			User:     os.Getenv("MYSQL_USER"),
			Password: os.Getenv("MYSQL_PASSWORD"),
			TLSMode:  os.Getenv("MYSQL_TLS_MODE"),
		},
		DatabaseURL:                  databaseURL,
		APIKeyEncryptionKey:          os.Getenv("API_KEY_ENCRYPTION_KEY"),
		APIKeyEncryptionPreviousKeys: splitNonEmpty(os.Getenv("API_KEY_ENCRYPTION_PREVIOUS_KEYS")),
	}, nil
}

func splitNonEmpty(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func mustGetEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("환경변수 %s 가 설정되지 않았습니다", key)
	}
	return v, nil
}
