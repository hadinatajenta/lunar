package crypto

import (
	"encoding/base64"
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt1 := GenerateSalt()
	salt2 := GenerateSalt()

	if len(salt1) != 32 {
		t.Fatalf("expected salt length of 32 hex chars, got %d", len(salt1))
	}
	if salt1 == salt2 {
		t.Fatalf("expected distinct salts, got identical values: %s", salt1)
	}
}

func TestPasswordHashingAndVerification(t *testing.T) {
	password := "SecretP@ssw0rd123"
	salt := GenerateSalt()

	hash := HashPassword(password, salt)
	if len(hash) != 64 {
		t.Fatalf("expected SHA-256 hex string of length 64, got %d", len(hash))
	}

	if !VerifyPassword(password, salt, hash) {
		t.Fatalf("expected password verification to succeed")
	}

	if VerifyPassword("WrongPassword", salt, hash) {
		t.Fatalf("expected verification to fail with wrong password")
	}

	anotherSalt := GenerateSalt()
	if VerifyPassword(password, anotherSalt, hash) {
		t.Fatalf("expected verification to fail with different salt")
	}
}

func TestAES256GCMRoundtrip(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "test-secret-token-value-42"

	encrypted, err := EncryptSecret(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}
	if encrypted == plaintext {
		t.Fatalf("ciphertext must not match plaintext")
	}

	decrypted, err := DecryptSecret(encrypted, key)
	if err != nil {
		t.Fatalf("failed to decrypt secret: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected decrypted text %s, got %s", plaintext, decrypted)
	}
}

func TestAES256GCMArbitraryKeyLength(t *testing.T) {
	passphrase := "short-passphrase"
	plaintext := "sample payload data"

	encrypted, err := EncryptSecret(plaintext, passphrase)
	if err != nil {
		t.Fatalf("failed to encrypt with arbitrary key: %v", err)
	}

	decrypted, err := DecryptSecret(encrypted, passphrase)
	if err != nil {
		t.Fatalf("failed to decrypt with arbitrary key: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("expected decrypted text %s, got %s", plaintext, decrypted)
	}
}

func TestAES256GCMEmptyPlaintext(t *testing.T) {
	key := "12345678901234567890123456789012"
	emptyText := ""

	encrypted, err := EncryptSecret(emptyText, key)
	if err != nil {
		t.Fatalf("failed to encrypt empty string: %v", err)
	}

	decrypted, err := DecryptSecret(encrypted, key)
	if err != nil {
		t.Fatalf("failed to decrypt empty ciphertext: %v", err)
	}
	if decrypted != emptyText {
		t.Fatalf("expected empty string, got %s", decrypted)
	}

	decryptedEmpty, err := DecryptSecret("", key)
	if err != nil {
		t.Fatalf("failed to decrypt empty string directly: %v", err)
	}
	if decryptedEmpty != "" {
		t.Fatalf("expected empty string, got %s", decryptedEmpty)
	}
}

func TestAES256GCMTamperedCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "super-confidential-atlassian-pat"

	encrypted, err := EncryptSecret(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	rawBytes, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}

	rawBytes[len(rawBytes)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(rawBytes)

	_, err = DecryptSecret(tampered, key)
	if err == nil {
		t.Fatalf("expected decryption error on tampered ciphertext, got nil")
	}
}

func TestAES256GCMInvalidKey(t *testing.T) {
	_, err := EncryptSecret("test", "")
	if err == nil {
		t.Fatalf("expected error on empty encryption key, got nil")
	}

	_, err = DecryptSecret("validbase64", "")
	if err == nil {
		t.Fatalf("expected error on empty decryption key, got nil")
	}
}

func TestAES256GCMInvalidCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"

	_, err := DecryptSecret("invalid-base64-!@#$", key)
	if err == nil {
		t.Fatalf("expected error on invalid base64, got nil")
	}

	shortData := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = DecryptSecret(shortData, key)
	if err == nil {
		t.Fatalf("expected error on ciphertext shorter than nonce, got nil")
	}
}
