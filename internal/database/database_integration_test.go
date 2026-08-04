package database

import (
	"os"
	"testing"

	"maple-discord-bot/internal/config"
)

func TestConnectMySQL(t *testing.T) {
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		t.Skip("MYSQL_TEST_HOST가 설정되지 않아 MySQL 통합 테스트를 건너뜁니다")
	}

	db, err := ConnectMySQL(config.MySQLConfig{
		Host:     host,
		Port:     os.Getenv("MYSQL_PORT"),
		Database: os.Getenv("MYSQL_DATABASE"),
		User:     os.Getenv("MYSQL_USER"),
		Password: os.Getenv("MYSQL_PASSWORD"),
		TLSMode:  os.Getenv("MYSQL_TLS_MODE"),
	})
	if err != nil {
		t.Fatalf("MySQL 연결 실패: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
}
