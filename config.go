package logger

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env        string
	Level      string
	Caller     bool
	Console    bool
	File       bool
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

func defaultConfig() Config {
	return Config{
		Env:        "production",
		Level:      "info",
		Caller:     false,
		Console:    true,
		File:       true,
		FilePath:   "./logs/app.log",
		MaxSizeMB:  100,
		MaxBackups: 30,
		MaxAgeDays: 30,
		Compress:   false,
	}
}

func LoadConfig() Config {
	_ = godotenv.Load()
	cfg := defaultConfig()
	if v := os.Getenv("LOGGER_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("LOGGER_LEVEL"); v != "" {
		cfg.Level = v
	}
	if v := os.Getenv("LOGGER_CALLER"); v != "" {
		cfg.Caller = parseBool(v, cfg.Caller)
	}
	if v := os.Getenv("LOGGER_CONSOLE"); v != "" {
		cfg.Console = parseBool(v, cfg.Console)
	}
	if v := os.Getenv("LOGGER_FILE"); v != "" {
		cfg.File = parseBool(v, cfg.File)
	}
	if v := os.Getenv("LOGGER_FILE_PATH"); v != "" {
		cfg.FilePath = v
	}
	if v := os.Getenv("LOGGER_MAX_SIZE_MB"); v != "" {
		cfg.MaxSizeMB = parseInt(v, cfg.MaxSizeMB)
	}
	if v := os.Getenv("LOGGER_MAX_BACKUPS"); v != "" {
		cfg.MaxBackups = parseInt(v, cfg.MaxBackups)
	}
	if v := os.Getenv("LOGGER_MAX_AGE_DAYS"); v != "" {
		cfg.MaxAgeDays = parseInt(v, cfg.MaxAgeDays)
	}
	if v := os.Getenv("LOGGER_COMPRESS"); v != "" {
		cfg.Compress = parseBool(v, cfg.Compress)
	}
	return cfg
}

func parseBool(s string, def bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func parseInt(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
