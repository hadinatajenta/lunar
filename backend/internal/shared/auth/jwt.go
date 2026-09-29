package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
)

type Claims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func GenerateToken(userID, email, secret string, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", sharedErrors.ErrInvalidKey
	}
	now := time.Now()
	hdr := header{
		Algorithm: "HS256",
		Type:      "JWT",
	}
	hdrBytes, err := json.Marshal(hdr)
	if err != nil {
		return "", err
	}
	claims := Claims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	hdrEncoded := base64.RawURLEncoding.EncodeToString(hdrBytes)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := hdrEncoded + "." + claimsEncoded

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	if len(secret) == 0 {
		return nil, sharedErrors.ErrInvalidKey
	}
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, sharedErrors.ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSignature := mac.Sum(nil)

	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, sharedErrors.ErrInvalidToken
	}

	if subtle.ConstantTimeCompare(actualSignature, expectedSignature) != 1 {
		return nil, sharedErrors.ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, sharedErrors.ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, sharedErrors.ErrInvalidToken
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, sharedErrors.ErrTokenExpired
	}

	if len(claims.UserID) == 0 {
		return nil, sharedErrors.ErrInvalidToken
	}

	return &claims, nil
}
