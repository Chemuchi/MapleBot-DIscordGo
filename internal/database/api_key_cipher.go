package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const encryptedAPIKeyVersion = "v1"

// APIKeyCipher는 현재 키와 이전 키들을 함께 보관해 안전한 키 교체를 지원합니다.
// 키 ID는 키 자체가 아니라 SHA-256 해시 일부이므로 DB에 비밀값이 노출되지 않습니다.
type APIKeyCipher struct {
	activeKeyID string
	keys        map[string]cipher.AEAD
}

// NewAPIKeyCipher는 Base64로 인코딩된 32바이트 AES-256 키를 구성합니다.
// previousKeys에 직전 키를 남겨두면 현재 키 변경 후에도 기존 데이터를 복호화할 수 있습니다.
func NewAPIKeyCipher(activeKey string, previousKeys []string) (*APIKeyCipher, error) {
	if strings.TrimSpace(activeKey) == "" {
		return nil, fmt.Errorf("API_KEY_ENCRYPTION_KEY가 설정되지 않았습니다")
	}

	c := &APIKeyCipher{keys: make(map[string]cipher.AEAD)}
	allKeys := append([]string{activeKey}, previousKeys...)
	for index, encodedKey := range allKeys {
		encodedKey = strings.TrimSpace(encodedKey)
		if encodedKey == "" {
			continue
		}
		key, err := decodeEncryptionKey(encodedKey)
		if err != nil {
			return nil, fmt.Errorf("API 키 암호화 키 %d: %w", index+1, err)
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("AES 초기화 실패: %w", err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("AES-GCM 초기화 실패: %w", err)
		}
		keyID := encryptionKeyID(key)
		c.keys[keyID] = aead
		if index == 0 {
			c.activeKeyID = keyID
		}
	}
	return c, nil
}

func decodeEncryptionKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return nil, fmt.Errorf("Base64 형식이 아닙니다: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("복호화한 키 길이가 %d바이트입니다 (32바이트 필요)", len(key))
	}
	return key, nil
}

func encryptionKeyID(key []byte) string {
	digest := sha256.Sum256(key)
	return hex.EncodeToString(digest[:8])
}

// Encrypt는 매번 새로운 nonce를 만들어 같은 API 키도 서로 다른 암호문으로 저장합니다.
func (c *APIKeyCipher) Encrypt(plaintext string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("API 키 암호화기가 초기화되지 않았습니다")
	}
	aead := c.keys[c.activeKeyID]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("암호화 nonce 생성 실패: %w", err)
	}
	sealed := aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, sealed...)
	return strings.Join([]string{
		encryptedAPIKeyVersion,
		c.activeKeyID,
		base64.RawStdEncoding.EncodeToString(payload),
	}, "."), nil
}

// Decrypt는 암호문을 복호화하고, 이전 키로 암호화된 값인지 함께 알려줍니다.
func (c *APIKeyCipher) Decrypt(encrypted string) (plaintext string, needsRotation bool, err error) {
	if c == nil {
		return "", false, fmt.Errorf("API 키 암호화기가 초기화되지 않았습니다")
	}
	parts := strings.Split(encrypted, ".")
	if len(parts) != 3 || parts[0] != encryptedAPIKeyVersion {
		return "", false, fmt.Errorf("지원하지 않는 API 키 암호문 형식입니다")
	}
	aead, ok := c.keys[parts[1]]
	if !ok {
		return "", false, fmt.Errorf("암호화에 사용된 키(%s)가 현재 키 목록에 없습니다", parts[1])
	}
	payload, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false, fmt.Errorf("API 키 암호문 디코딩 실패: %w", err)
	}
	if len(payload) < aead.NonceSize() {
		return "", false, fmt.Errorf("API 키 암호문 길이가 올바르지 않습니다")
	}
	nonce, ciphertext := payload[:aead.NonceSize()], payload[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", false, fmt.Errorf("API 키 복호화 또는 무결성 검증 실패: %w", err)
	}
	return string(plain), parts[1] != c.activeKeyID, nil
}
