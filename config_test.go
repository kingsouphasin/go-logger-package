package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "info", cfg.Level)
	assert.False(t, cfg.Caller)
	assert.True(t, cfg.Console)
	assert.True(t, cfg.File)
	assert.Equal(t, "./logs/app.log", cfg.FilePath)
	assert.Equal(t, 100, cfg.MaxSizeMB)
	assert.Equal(t, 30, cfg.MaxBackups)
	assert.Equal(t, 30, cfg.MaxAgeDays)
	assert.False(t, cfg.Compress)
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("LOGGER_ENV", "development")
	t.Setenv("LOGGER_LEVEL", "debug")
	t.Setenv("LOGGER_CALLER", "true")
	t.Setenv("LOGGER_CONSOLE", "false")
	t.Setenv("LOGGER_FILE", "false")
	t.Setenv("LOGGER_FILE_PATH", "/tmp/test.log")
	t.Setenv("LOGGER_MAX_SIZE_MB", "50")
	t.Setenv("LOGGER_MAX_BACKUPS", "5")
	t.Setenv("LOGGER_MAX_AGE_DAYS", "7")
	t.Setenv("LOGGER_COMPRESS", "true")

	cfg := LoadConfig()
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "debug", cfg.Level)
	assert.True(t, cfg.Caller)
	assert.False(t, cfg.Console)
	assert.False(t, cfg.File)
	assert.Equal(t, "/tmp/test.log", cfg.FilePath)
	assert.Equal(t, 50, cfg.MaxSizeMB)
	assert.Equal(t, 5, cfg.MaxBackups)
	assert.Equal(t, 7, cfg.MaxAgeDays)
	assert.True(t, cfg.Compress)
}

func TestInvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("LOGGER_MAX_SIZE_MB", "notanumber")
	cfg := LoadConfig()
	assert.Equal(t, 100, cfg.MaxSizeMB)
}

func TestInvalidBoolFallsBackToDefault(t *testing.T) {
	t.Setenv("LOGGER_CALLER", "notabool")
	cfg := LoadConfig()
	assert.False(t, cfg.Caller)
}
