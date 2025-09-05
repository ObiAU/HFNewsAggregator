package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	NewsAPIKey         string
	OpenAIAPIKey       string
	TelegramToken      string
	TelegramWebhookURL string
	BatchSize          int
	ProcessingInterval time.Duration
	CacheRetention     time.Duration
	ServerPort         string
	LogLevel           string
}

func Load() *Config {
	return &Config{
		NewsAPIKey:         getEnv("NEWS_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		TelegramToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhookURL: getEnv("TELEGRAM_WEBHOOK_URL", ""),
		BatchSize:          getEnvAsInt("BATCH_SIZE", 10),
		ProcessingInterval: getEnvAsDuration("PROCESSING_INTERVAL", 30*time.Second),
		CacheRetention:     getEnvAsDuration("CACHE_RETENTION", 24*time.Hour),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
