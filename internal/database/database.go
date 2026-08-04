package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"maple-discord-bot/internal/config"
)

const (
	connectionTimeout = 5 * time.Second
	pingTimeout       = 10 * time.Second
)

// ConnectMySQL : 개별 설정으로 안전하게 DSN을 만들고 MySQL 연결을 확인합니다.
func ConnectMySQL(settings config.MySQLConfig) (*sql.DB, error) {
	if settings.Host == "" || settings.Port == "" || settings.Database == "" || settings.User == "" || settings.Password == "" {
		return nil, fmt.Errorf("MYSQL_HOST, MYSQL_PORT, MYSQL_DATABASE, MYSQL_USER, MYSQL_PASSWORD를 모두 설정해야 합니다")
	}

	tlsMode := settings.TLSMode
	if tlsMode == "" {
		tlsMode = "false"
	}

	driverConfig := mysql.Config{
		User:                 settings.User,
		Passwd:               settings.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(settings.Host, settings.Port),
		DBName:               settings.Database,
		ParseTime:            true,
		Timeout:              connectionTimeout,
		ReadTimeout:          10 * time.Second,
		WriteTimeout:         10 * time.Second,
		TLSConfig:            tlsMode,
		AllowNativePasswords: true,
	}
	dsn := driverConfig.FormatDSN()

	return openAndPing("mysql", dsn)
}

// ConnectPostgres : 운영 전환 전 기존 Supabase 연결을 유지하기 위한 임시 호환 경로입니다.
func ConnectPostgres(connStr string) (*sql.DB, error) {
	if connStr == "" {
		return nil, fmt.Errorf("데이터베이스 연결 URL이 비어 있습니다")
	}
	return openAndPing("postgres", connStr)
}

func openAndPing(driverName, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("DB 드라이버 초기화 실패: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("DB 연결 확인(Ping) 실패: %w", err)
	}

	return db, nil
}
