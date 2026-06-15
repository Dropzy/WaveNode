package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	MusicPath         string
	Environment       string
	CORSOrigins       []string
	AllowRegistration bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Port:        getEnv("SERVER_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/musicserver?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		MusicPath:   getEnv("MUSIC_PATH", ""),
		Environment: strings.ToLower(getEnv("APP_ENV", "development")),
	}

	defaultOrigins := "http://localhost:5173,http://127.0.0.1:5173"
	cfg.CORSOrigins = splitCSV(getEnv("CORS_ALLOWED_ORIGINS", defaultOrigins))

	defaultRegistration := cfg.Environment != "production"
	allowRegistration, err := strconv.ParseBool(getEnv("ALLOW_REGISTRATION", strconv.FormatBool(defaultRegistration)))
	if err != nil {
		return nil, fmt.Errorf("ALLOW_REGISTRATION must be true or false")
	}
	cfg.AllowRegistration = allowRegistration

	if cfg.Environment == "production" {
		if len(cfg.JWTSecret) < 32 || cfg.JWTSecret == "your-secret-key-change-this-in-production" {
			return nil, fmt.Errorf("JWT_SECRET must be a unique value of at least 32 characters in production")
		}
		if cfg.DatabaseURL == "postgres://localhost/musicserver?sslmode=disable" {
			return nil, fmt.Errorf("DATABASE_URL must be configured in production")
		}
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitCSV(value string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	return values
}
