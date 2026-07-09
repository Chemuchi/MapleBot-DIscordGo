package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Connect : 제공된 연결 URL로 PostgreSQL(Supabase) 데이터베이스에 접속을 테스트하고 객체를 반환합니다.
func Connect(connStr string) (*sql.DB, error) {
	if connStr == "" {
		return nil, fmt.Errorf("데이터베이스 연결 URL이 비어 있습니다")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("DB 드라이버 초기화 실패: %w", err)
	}

	// 실제 연결 상태 확인
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("DB 연결 확인(Ping) 실패: %w", err)
	}

	return db, nil
}
