package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ensureDevTestUser creates or updates a predictable local account for live testing.
// It is strictly opt-in and only runs when DEV_SEED_TEST_USER=true.
func (db *DB) ensureDevTestUser() error {
	if os.Getenv("DEV_SEED_TEST_USER") != "true" {
		return nil
	}
	if os.Getenv("APP_ENV") == "production" {
		return fmt.Errorf("DEV_SEED_TEST_USER cannot be enabled in production")
	}

	username := getDevSeedEnv("DEV_TEST_USERNAME", "live-test-admin")
	email := getDevSeedEnv("DEV_TEST_EMAIL", "live-test-admin@example.com")
	password := getDevSeedEnv("DEV_TEST_PASSWORD", "admin123")
	role := getDevSeedEnv("DEV_TEST_ROLE", "admin")

	if role != "admin" && role != "user" {
		return fmt.Errorf("DEV_TEST_ROLE must be either admin or user")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash live test password: %v", err)
	}

	now := time.Now()
	id := fmt.Sprintf("user_%d", now.UnixNano())

	query := `
		INSERT INTO users (id, username, email, role, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (username) DO UPDATE SET
			email = EXCLUDED.email,
			role = EXCLUDED.role,
			password = EXCLUDED.password,
			updated_at = EXCLUDED.updated_at
	`

	if _, err := db.conn.Exec(query, id, username, email, role, string(hashedPassword), now, now); err != nil {
		return fmt.Errorf("failed to seed live test user: %v", err)
	}

	log.Printf("Live test user ready (username: %s)", username)
	return nil
}

func getDevSeedEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
