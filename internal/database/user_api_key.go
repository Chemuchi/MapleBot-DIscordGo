package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// UserAPIKeyStore는 디스코드 사용자별 넥슨 Open API 키를 저장합니다.
type UserAPIKeyStore struct {
	db      *sql.DB
	dialect string
	cipher  *APIKeyCipher
}

func NewUserAPIKeyStore(db *sql.DB, apiKeyCipher *APIKeyCipher) *UserAPIKeyStore {
	dialect := "mysql"
	if db != nil && strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "pq") {
		dialect = "postgres"
	}
	return &UserAPIKeyStore{db: db, dialect: dialect, cipher: apiKeyCipher}
}

// EnsureSchema는 API 키 저장 테이블이 없으면 생성합니다.
func (s *UserAPIKeyStore) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("데이터베이스가 연결되어 있지 않습니다")
	}
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_api_keys (
			discord_user_id VARCHAR(32) PRIMARY KEY,
			api_key VARCHAR(512) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("user_api_keys 테이블 생성 실패: %w", err)
	}
	return s.migrateEncryption(ctx)
}

// migrateEncryption은 평문 및 이전 키 암호문을 시작 시 현재 키 암호문으로 교체합니다.
func (s *UserAPIKeyStore) migrateEncryption(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "SELECT discord_user_id, api_key FROM user_api_keys")
	if err != nil {
		return fmt.Errorf("기존 사용자 API 키 확인 실패: %w", err)
	}
	type record struct{ userID, apiKey string }
	var recordsToEncrypt []record
	for rows.Next() {
		var item record
		if err := rows.Scan(&item.userID, &item.apiKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("기존 사용자 API 키 읽기 실패: %w", err)
		}
		if strings.HasPrefix(item.apiKey, encryptedAPIKeyVersion+".") {
			plaintext, needsRotation, err := s.cipher.Decrypt(item.apiKey)
			if err != nil {
				_ = rows.Close()
				return fmt.Errorf("기존 사용자 API 키 복호화 실패: %w", err)
			}
			if needsRotation {
				recordsToEncrypt = append(recordsToEncrypt, record{userID: item.userID, apiKey: plaintext})
			}
		} else {
			recordsToEncrypt = append(recordsToEncrypt, item)
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("기존 사용자 API 키 조회 종료 실패: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("기존 사용자 API 키 조회 실패: %w", err)
	}

	for _, item := range recordsToEncrypt {
		encrypted, err := s.cipher.Encrypt(item.apiKey)
		if err != nil {
			return fmt.Errorf("기존 사용자 API 키 암호화 실패: %w", err)
		}
		if err := s.saveEncrypted(ctx, item.userID, encrypted); err != nil {
			return fmt.Errorf("기존 사용자 API 키 암호화 저장 실패: %w", err)
		}
	}
	return nil
}

// Save는 사용자 키를 새로 저장하거나 기존 값을 교체합니다.
func (s *UserAPIKeyStore) Save(ctx context.Context, discordUserID, apiKey string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("데이터베이스가 연결되어 있지 않습니다")
	}

	encrypted, err := s.cipher.Encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("사용자 API 키 암호화 실패: %w", err)
	}
	return s.saveEncrypted(ctx, discordUserID, encrypted)
}

func (s *UserAPIKeyStore) saveEncrypted(ctx context.Context, discordUserID, encrypted string) error {
	query := `INSERT INTO user_api_keys (discord_user_id, api_key)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE api_key = VALUES(api_key), updated_at = CURRENT_TIMESTAMP`
	if s.dialect == "postgres" {
		query = `INSERT INTO user_api_keys (discord_user_id, api_key)
			VALUES ($1, $2)
			ON CONFLICT (discord_user_id) DO UPDATE
			SET api_key = EXCLUDED.api_key, updated_at = CURRENT_TIMESTAMP`
	}

	if _, err := s.db.ExecContext(ctx, query, discordUserID, encrypted); err != nil {
		return fmt.Errorf("사용자 API 키 저장 실패: %w", err)
	}
	return nil
}

// Find는 사용자 키를 반환합니다. 등록되지 않은 사용자는 sql.ErrNoRows를 반환합니다.
func (s *UserAPIKeyStore) Find(ctx context.Context, discordUserID string) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("데이터베이스가 연결되어 있지 않습니다")
	}
	placeholder := "?"
	if s.dialect == "postgres" {
		placeholder = "$1"
	}
	var encrypted string
	err := s.db.QueryRowContext(ctx,
		"SELECT api_key FROM user_api_keys WHERE discord_user_id = "+placeholder,
		discordUserID,
	).Scan(&encrypted)
	if err != nil {
		return "", err
	}

	apiKey, needsRotation, err := s.cipher.Decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("사용자 API 키 복호화 실패: %w", err)
	}
	if needsRotation {
		// 읽기는 성공시킨 뒤 최선 노력으로 현재 키 암호문으로 교체합니다.
		if rotated, encryptErr := s.cipher.Encrypt(apiKey); encryptErr == nil {
			_ = s.saveEncrypted(ctx, discordUserID, rotated)
		}
	}
	return apiKey, nil
}

// Delete는 해당 디스코드 사용자의 행 전체를 삭제합니다.
// 반환값은 실제로 삭제된 행이 있었는지를 나타냅니다.
func (s *UserAPIKeyStore) Delete(ctx context.Context, discordUserID string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("데이터베이스가 연결되어 있지 않습니다")
	}
	placeholder := "?"
	if s.dialect == "postgres" {
		placeholder = "$1"
	}
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM user_api_keys WHERE discord_user_id = "+placeholder,
		discordUserID,
	)
	if err != nil {
		return false, fmt.Errorf("사용자 정보 삭제 실패: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("사용자 정보 삭제 결과 확인 실패: %w", err)
	}
	return rowsAffected > 0, nil
}
