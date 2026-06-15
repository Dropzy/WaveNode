package config

import (
	"strings"
	"testing"
)

func TestProductionRequiresStrongJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "short")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET validation error, got %v", err)
	}
}

func TestProductionDefaultsRegistrationOff(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "a-unique-production-secret-that-is-long-enough")
	t.Setenv("ALLOW_REGISTRATION", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://music.example, https://admin.example")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.AllowRegistration {
		t.Fatal("expected public registration to default off in production")
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "https://music.example" {
		t.Fatalf("unexpected CORS origins: %#v", cfg.CORSOrigins)
	}
}

func TestProductionRejectsShortSetupToken(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "a-unique-production-secret-that-is-long-enough")
	t.Setenv("SETUP_TOKEN", "too-short")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "SETUP_TOKEN") {
		t.Fatalf("expected SETUP_TOKEN validation error, got %v", err)
	}
}
