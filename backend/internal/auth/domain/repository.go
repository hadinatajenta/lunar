package domain

import "context"

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

type VaultRepository interface {
	SaveSecrets(ctx context.Context, secrets *UserSecrets) error
	GetSecretsByUserID(ctx context.Context, userID string) (*UserSecrets, error)
}
