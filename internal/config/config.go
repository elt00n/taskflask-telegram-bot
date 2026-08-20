// Package config отвечает за настройки приложения.
package config

import "os"

const defaultEnvironment = "development"

// Config объединяет настройки, которые приложение получает из окружения.
// Секретные значения нельзя выводить в логи или сохранять в Git.
type Config struct {
	Environment      string
	TelegramBotToken string
	DatabaseURL      string
}

// Load читает настройки процесса и возвращает готовую конфигурацию.
// Проверку обязательных полей добавим тогда, когда подключим Telegram и PostgreSQL.
func Load() Config {
	return Config{
		Environment:      envOrDefault("APP_ENV", defaultEnvironment),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
