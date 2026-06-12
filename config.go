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
		cfg.Caller = parseBool(v, false)
	}
	if v := os.Getenv("LOGGER_CONSOLE"); v != "" {
		cfg.Console = parseBool(v, true)
	}
	if v := os.Getenv("LOGGER_FILE"); v != "" {
		cfg.File = parseBool(v, true)
	}
	if v := os.Getenv("LOGGER_FILE_PATH"); v != "" {
		cfg.FilePath = v
	}
	if v := os.Getenv("LOGGER_MAX_SIZE_MB"); v != "" {
		cfg.MaxSizeMB = parseInt(v, 100)
	}
	if v := os.Getenv("LOGGER_MAX_BACKUPS"); v != "" {
		cfg.MaxBackups = parseInt(v, 30)
	}
	if v := os.Getenv("LOGGER_MAX_AGE_DAYS"); v != "" {
		cfg.MaxAgeDays = parseInt(v, 30)
	}
	if v := os.Getenv("LOGGER_COMPRESS"); v != "" {
		cfg.Compress = parseBool(v, false)
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
