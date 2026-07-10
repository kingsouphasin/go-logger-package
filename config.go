package logger

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Env sets the output format. "development" prints colored human-readable
	// logs; "production" prints JSON. Default: "production".
	Env string

	// Level is the minimum log level to emit: "debug", "info", "warn", "error",
	// or "fatal". Messages below this level are silently dropped. Default: "info".
	Level string

	// Caller adds a "caller" field (file:line) to every log entry. Default: false.
	Caller bool

	// Console enables writing log output to stdout. Default: true.
	Console bool

	// File enables writing log output to a rotating file. Default: true.
	File bool

	// FilePath is the path of the log file. Default: "./logs/app.log".
	FilePath string

	// MaxSizeMB is the maximum size in megabytes a log file can reach before it
	// is rotated. Default: 100.
	MaxSizeMB int

	// MaxBackups is the maximum number of old rotated log files to keep.
	// Older files are deleted once this limit is exceeded. Default: 30.
	MaxBackups int

	// MaxAgeDays is the maximum number of days to retain old log files.
	// Files older than this are deleted during the next rotation. Default: 30.
	MaxAgeDays int

	// Compress gzip-compresses rotated log files to save disk space. Default: true.
	Compress bool

	// LogBody enables logging of HTTP request and response body content by the
	// middleware. Bodies are captured only for text/JSON content types, JSON
	// sensitive keys are redacted, and content is truncated to MaxBodyBytes.
	// Default: false.
	LogBody bool

	// MaxBodyBytes is the maximum number of bytes of a request or response body
	// to log when LogBody is enabled. Longer bodies are truncated. Default: 4096.
	MaxBodyBytes int
}

func defaultConfig() Config {
	return Config{
		Env:          "production",
		Level:        "info",
		Caller:       false,
		Console:      true,
		File:         true,
		FilePath:     "./logs/app.log",
		MaxSizeMB:    100,
		MaxBackups:   30,
		MaxAgeDays:   30,
		Compress:     true,
		LogBody:      false,
		MaxBodyBytes: 4096,
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
	if v := os.Getenv("LOGGER_LOG_BODY"); v != "" {
		cfg.LogBody = parseBool(v, cfg.LogBody)
	}
	if v := os.Getenv("LOGGER_MAX_BODY_BYTES"); v != "" {
		cfg.MaxBodyBytes = parseInt(v, cfg.MaxBodyBytes)
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
