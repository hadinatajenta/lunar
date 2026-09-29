package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"lunar/backend/internal/auth/domain"
	"lunar/backend/internal/shared/auth"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember,omitempty"`
}

type AuthResponse struct {
	Token string                   `json:"token"`
	User  domain.UserPublicProfile `json:"user"`
}

type AuthService struct {
	userRepo      domain.UserRepository
	vaultRepo     domain.VaultRepository
	encryptionKey string
	jwtSecret     string
	jwtTTL        time.Duration
}

func NewAuthService(
	userRepo domain.UserRepository,
	vaultRepo domain.VaultRepository,
	encryptionKey string,
	jwtSecret string,
	jwtTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		vaultRepo:     vaultRepo,
		encryptionKey: encryptionKey,
		jwtSecret:     jwtSecret,
		jwtTTL:        jwtTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	fullName := strings.TrimSpace(req.FullName)

	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: valid email is required", sharedErrors.ErrBadRequest)
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", sharedErrors.ErrBadRequest)
	}
	if fullName == "" {
		return nil, fmt.Errorf("%w: full name is required", sharedErrors.ErrBadRequest)
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, sharedErrors.ErrConflict
	}
	if err != nil && !errors.Is(err, sharedErrors.ErrNotFound) {
		return nil, err
	}

	salt := crypto.GenerateSalt()
	passwordHash := crypto.HashPassword(req.Password, salt)
	now := time.Now().UTC()

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		FullName:     fullName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(user.ID, user.Email, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToPublic(),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, fmt.Errorf("%w: email and password are required", sharedErrors.ErrBadRequest)
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, sharedErrors.ErrUnauthorized
	}

	if !crypto.VerifyPassword(req.Password, user.Salt, user.PasswordHash) {
		return nil, sharedErrors.ErrUnauthorized
	}

	ttl := s.jwtTTL
	if req.Remember {
		ttl = 30 * 24 * time.Hour
	}

	token, err := auth.GenerateToken(user.ID, user.Email, s.jwtSecret, ttl)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToPublic(),
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*domain.UserPublicProfile, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	profile := user.ToPublic()
	return &profile, nil
}

func (s *AuthService) SaveUserSecrets(ctx context.Context, userID string, input domain.SaveSecretsInput) error {
	if _, err := s.userRepo.GetUserByID(ctx, userID); err != nil {
		return err
	}

	existing, err := s.vaultRepo.GetSecretsByUserID(ctx, userID)
	var secrets domain.UserSecrets
	if err == nil && existing != nil {
		secrets = *existing
	} else if err != nil && errors.Is(err, sharedErrors.ErrNotFound) {
		secrets = domain.UserSecrets{
			UserID: userID,
		}
	} else {
		return err
	}

	secrets.UpdatedAt = time.Now().UTC()

	if input.JiraPAT != "" {
		enc, err := crypto.EncryptSecret(input.JiraPAT, s.encryptionKey)
		if err != nil {
			return err
		}
		secrets.JiraPATEnc = enc
	}
	if input.JiraUsername != "" {
		secrets.JiraUsername = input.JiraUsername
	}

	if input.BitbucketPAT != "" {
		enc, err := crypto.EncryptSecret(input.BitbucketPAT, s.encryptionKey)
		if err != nil {
			return err
		}
		secrets.BitbucketPATEnc = enc
	}
	if input.BitbucketUsername != "" {
		secrets.BitbucketUsername = input.BitbucketUsername
	}

	if input.ConfluencePAT != "" {
		enc, err := crypto.EncryptSecret(input.ConfluencePAT, s.encryptionKey)
		if err != nil {
			return err
		}
		secrets.ConfluencePATEnc = enc
	}

	if input.AIKeys != nil {
		currentAIKeys := make(map[string]string)
		if secrets.AIKeysJSONEnc != "" {
			decryptedJSON, err := crypto.DecryptSecret(secrets.AIKeysJSONEnc, s.encryptionKey)
			if err == nil && decryptedJSON != "" {
				_ = json.Unmarshal([]byte(decryptedJSON), &currentAIKeys)
			}
		}
		for key, val := range input.AIKeys {
			if strings.TrimSpace(val) != "" {
				currentAIKeys[key] = strings.TrimSpace(val)
			}
		}
		marshaled, err := json.Marshal(currentAIKeys)
		if err != nil {
			return err
		}
		encryptedAIKeys, err := crypto.EncryptSecret(string(marshaled), s.encryptionKey)
		if err != nil {
			return err
		}
		secrets.AIKeysJSONEnc = encryptedAIKeys
	}

	return s.vaultRepo.SaveSecrets(ctx, &secrets)
}

func (s *AuthService) GetRedactedSecrets(ctx context.Context, userID string) (*domain.RedactedSecrets, error) {
	secrets, err := s.vaultRepo.GetSecretsByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			return &domain.RedactedSecrets{
				UserID:                userID,
				ConfiguredAIProviders: []string{},
			}, nil
		}
		return nil, err
	}

	redacted := &domain.RedactedSecrets{
		UserID:                secrets.UserID,
		HasJiraPAT:            secrets.JiraPATEnc != "",
		JiraUsername:          secrets.JiraUsername,
		HasBitbucketPAT:       secrets.BitbucketPATEnc != "",
		BitbucketUsername:     secrets.BitbucketUsername,
		HasConfluencePAT:      secrets.ConfluencePATEnc != "",
		HasAIKeys:             secrets.AIKeysJSONEnc != "",
		ConfiguredAIProviders: make([]string, 0),
		UpdatedAt:             secrets.UpdatedAt,
	}

	if secrets.AIKeysJSONEnc != "" {
		decryptedJSON, err := crypto.DecryptSecret(secrets.AIKeysJSONEnc, s.encryptionKey)
		if err == nil && decryptedJSON != "" {
			var keysMap map[string]string
			if err := json.Unmarshal([]byte(decryptedJSON), &keysMap); err == nil {
				for provider, val := range keysMap {
					if strings.TrimSpace(val) != "" {
						redacted.ConfiguredAIProviders = append(redacted.ConfiguredAIProviders, provider)
					}
				}
				sort.Strings(redacted.ConfiguredAIProviders)
			}
		}
	}

	return redacted, nil
}

func (s *AuthService) GetDecryptedSecrets(ctx context.Context, userID string) (*domain.DecryptedSecrets, error) {
	secrets, err := s.vaultRepo.GetSecretsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	decrypted := &domain.DecryptedSecrets{
		UserID:            secrets.UserID,
		JiraUsername:      secrets.JiraUsername,
		BitbucketUsername: secrets.BitbucketUsername,
		AIKeys:            make(map[string]string),
		UpdatedAt:         secrets.UpdatedAt,
	}

	if secrets.JiraPATEnc != "" {
		pat, err := crypto.DecryptSecret(secrets.JiraPATEnc, s.encryptionKey)
		if err != nil {
			return nil, err
		}
		decrypted.JiraPAT = pat
	}

	if secrets.BitbucketPATEnc != "" {
		pat, err := crypto.DecryptSecret(secrets.BitbucketPATEnc, s.encryptionKey)
		if err != nil {
			return nil, err
		}
		decrypted.BitbucketPAT = pat
	}

	if secrets.ConfluencePATEnc != "" {
		pat, err := crypto.DecryptSecret(secrets.ConfluencePATEnc, s.encryptionKey)
		if err != nil {
			return nil, err
		}
		decrypted.ConfluencePAT = pat
	}

	if secrets.AIKeysJSONEnc != "" {
		decryptedJSON, err := crypto.DecryptSecret(secrets.AIKeysJSONEnc, s.encryptionKey)
		if err == nil && decryptedJSON != "" {
			var keysMap map[string]string
			if err := json.Unmarshal([]byte(decryptedJSON), &keysMap); err == nil {
				decrypted.AIKeys = keysMap
			}
		}
	}

	return decrypted, nil
}
