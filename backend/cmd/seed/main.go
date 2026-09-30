package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"lunar/backend/internal/auth/domain"
	"lunar/backend/internal/auth/infrastructure"
	"lunar/backend/internal/shared/crypto"
	"lunar/backend/internal/shared/database"
)

func main() {
	dbPath := "data/lunar.db"
	if v := os.Getenv("DB_PATH"); v != "" {
		dbPath = v
	}

	email := os.Args[1]
	password := os.Args[2]
	fullName := os.Args[3]

	db, err := database.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := infrastructure.NewSQLiteUserRepository(db)
	ctx := context.Background()

	existing, err := repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		log.Fatalf("user with email %q already exists (id=%s)", email, existing.ID)
	}

	salt := crypto.GenerateSalt()
	passwordHash := crypto.HashPassword(password, salt)
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

	if err := repo.CreateUser(ctx, user); err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("user created: id=%s email=%s full_name=%s\n", user.ID, user.Email, user.FullName)
}
