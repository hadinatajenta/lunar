package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

var (
	ErrEmptyKey       = errors.New("empty encryption key")
	ErrCiphertextShort = errors.New("ciphertext too short")
)

func GenerateSalt() string {
	bytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func HashPassword(password, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	hasher.Write([]byte(":"))
	hasher.Write([]byte(salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

func VerifyPassword(password, salt, expectedHash string) bool {
	computedHash := HashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(expectedHash)) == 1
}

func deriveAESKey(key string) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	if len(key) == 32 {
		return []byte(key), nil
	}
	hash := sha256.Sum256([]byte(key))
	return hash[:], nil
}

func EncryptSecret(plaintext, key string) (string, error) {
	aesKey, err := deriveAESKey(key)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func DecryptSecret(ciphertext, key string) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	aesKey, err := deriveAESKey(key)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextShort
	}
	nonce := data[:nonceSize]
	rawCiphertext := data[nonceSize:]
	plaintextBytes, err := gcm.Open(nil, nonce, rawCiphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintextBytes), nil
}
