package internal

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	CloudflareAccountID string
	CloudflareAPIKey    string
	LogLevel            slog.Leveler
	RunMigrations       bool
}

func LoadConfig() Config {

	envLevel := os.Getenv("LOG_LEVEL")
	runMigrationsEnv := strings.ToLower(os.Getenv("RUN_MIGRATIONS"))
	logLevel := slog.LevelDebug

	switch envLevel {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	}

	return Config{
		CloudflareAccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		CloudflareAPIKey:    os.Getenv("CLOUDFLARE_API_KEY"),
		LogLevel:            logLevel,
		RunMigrations:       runMigrationsEnv == "true",
	}
}
