package database

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testEncryptionKey(fill byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(fill), 32)))
}

func TestAPIKeyCipherRoundTripUsesRandomNonce(t *testing.T) {
	cipher, err := NewAPIKeyCipher(testEncryptionKey('a'), nil)
	if err != nil {
		t.Fatalf("NewAPIKeyCipher() error = %v", err)
	}

	first, err := cipher.Encrypt("nexon-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	second, err := cipher.Encrypt("nexon-secret")
	if err != nil {
		t.Fatalf("Encrypt() second error = %v", err)
	}
	if first == second {
		t.Fatal("같은 평문을 암호화한 결과가 같습니다; nonce가 매번 달라야 합니다")
	}
	if strings.Contains(first, "nexon-secret") {
		t.Fatal("암호문에 평문 API 키가 포함되어 있습니다")
	}

	plaintext, needsRotation, err := cipher.Decrypt(first)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plaintext != "nexon-secret" || needsRotation {
		t.Fatalf("Decrypt() = (%q, %v), want (%q, false)", plaintext, needsRotation, "nexon-secret")
	}
}

func TestAPIKeyCipherKeyRotation(t *testing.T) {
	oldKey := testEncryptionKey('o')
	newKey := testEncryptionKey('n')
	oldCipher, err := NewAPIKeyCipher(oldKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := oldCipher.Encrypt("existing-api-key")
	if err != nil {
		t.Fatal(err)
	}

	rotatedCipher, err := NewAPIKeyCipher(newKey, []string{oldKey})
	if err != nil {
		t.Fatal(err)
	}
	plaintext, needsRotation, err := rotatedCipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("이전 키를 포함한 복호화 실패: %v", err)
	}
	if plaintext != "existing-api-key" || !needsRotation {
		t.Fatalf("Decrypt() = (%q, %v), want (%q, true)", plaintext, needsRotation, "existing-api-key")
	}

	withoutOldKey, err := NewAPIKeyCipher(newKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := withoutOldKey.Decrypt(encrypted); err == nil {
		t.Fatal("이전 키 없이 기존 암호문 복호화가 성공했습니다")
	}
}

func TestAPIKeyCipherRejectsTamperedCiphertext(t *testing.T) {
	cipher, err := NewAPIKeyCipher(testEncryptionKey('a'), nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("nexon-secret")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(encrypted, ".")
	payload, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	payload[len(payload)-1] ^= 1
	parts[2] = base64.RawStdEncoding.EncodeToString(payload)
	tampered := strings.Join(parts, ".")
	if _, _, err := cipher.Decrypt(tampered); err == nil {
		t.Fatal("변조된 암호문의 무결성 검증이 성공했습니다")
	}
}

func TestAPIKeyCipherRejectsInvalidKey(t *testing.T) {
	if _, err := NewAPIKeyCipher("too-short", nil); err == nil {
		t.Fatal("잘못된 암호화 키를 허용했습니다")
	}
}
