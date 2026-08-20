// Package config отвечает за настройки приложения.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultEnvironment = "development"
	defaultTimezone    = "Europe/Moscow"
	defaultDatabaseURL = "postgres://taskflask:taskflask@localhost:5432/taskflask?sslmode=disable"
)

var ErrTelegramBotTokenRequired = errors.New("TELEGRAM_BOT_TOKEN is required")

// Config объединяет настройки, которые приложение получает из окружения.
// Секретные значения нельзя выводить в логи или сохранять в Git.
type Config struct {
	Environment      string
	DefaultTimezone  string
	TelegramBotToken string
	DatabaseURL      string
}

// Load загружает локальный .env, читает окружение и проверяет обязательные поля.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	config := Config{
		Environment:      envOrDefault("APP_ENV", defaultEnvironment),
		DefaultTimezone:  envOrDefault("DEFAULT_TIMEZONE", defaultTimezone),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:      envOrDefault("DATABASE_URL", defaultDatabaseURL),
	}
	if strings.TrimSpace(config.TelegramBotToken) == "" {
		return Config{}, ErrTelegramBotTokenRequired
	}
	if _, err := time.LoadLocation(config.DefaultTimezone); err != nil {
		return Config{}, fmt.Errorf("invalid DEFAULT_TIMEZONE %q: %w", config.DefaultTimezone, err)
	}

	return config, nil
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
