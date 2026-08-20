package config_test

import (
	"errors"
	"testing"

	"github.com/elt00n/taskflask-telegram-bot/internal/config"
)

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("DEFAULT_TIMEZONE", "UTC")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DATABASE_URL", "postgres://example")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if got.Environment != "test" {
		t.Errorf("Environment = %q, want %q", got.Environment, "test")
	}
	if got.DefaultTimezone != "UTC" {
		t.Errorf("DefaultTimezone = %q, want %q", got.DefaultTimezone, "UTC")
	}
	if got.TelegramBotToken != "test-token" {
		t.Error("TelegramBotToken was not loaded from the environment")
	}
	if got.DatabaseURL != "postgres://example" {
		t.Errorf("DatabaseURL = %q, want %q", got.DatabaseURL, "postgres://example")
	}
}

func TestLoadUsesLocalDatabaseByDefault(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DATABASE_URL", "")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if got.DatabaseURL == "" {
		t.Fatal("DatabaseURL must have a local development default")
	}
}

func TestLoadRequiresTelegramToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	_, err := config.Load()
	if !errors.Is(err, config.ErrTelegramBotTokenRequired) {
		t.Fatalf("Load() error = %v, want %v", err, config.ErrTelegramBotTokenRequired)
	}
}

func TestLoadRejectsInvalidTimezone(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DEFAULT_TIMEZONE", "Mars/Olympus")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must reject an invalid timezone")
	}
}
