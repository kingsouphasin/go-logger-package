package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewConsoleOnly(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
	_ = log.Sync()
}

func TestNewDevelopmentMode(t *testing.T) {
	cfg := Config{Env: "development", Level: "debug", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
	_ = log.Sync()
}

func TestNewFileOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Env:        "production",
		Level:      "info",
		Console:    false,
		File:       true,
		FilePath:   filepath.Join(dir, "sub", "app.log"),
		MaxSizeMB:  10,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   false,
	}
	log, err := New(cfg)
	require.NoError(t, err)
	log.Info("test message")
	_ = log.Sync()

	_, err = os.Stat(cfg.FilePath)
	assert.NoError(t, err, "log file should be created")
}

func TestNewBothOutputs(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Env:      "production",
		Level:    "info",
		Console:  true,
		File:     true,
		FilePath: filepath.Join(dir, "app.log"),
	}
	log, err := New(cfg)
	require.NoError(t, err)
	log.Info("both outputs")
	_ = log.Sync()
}

func TestNewInvalidLevelFallsBackToInfo(t *testing.T) {
	cfg := Config{Env: "production", Level: "badlevel", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
	_ = log.Sync()
}

func TestSetLevel(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)

	assert.NoError(t, log.SetLevel("debug"))
	assert.NoError(t, log.SetLevel("warn"))
	assert.Error(t, log.SetLevel("invalid"))
	_ = log.Sync()
}

func TestWithChildLogger(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)

	child := log.With(zap.String("service", "auth"))
	assert.NotNil(t, child)
	child.Info("child logger works")
	_ = log.Sync()
}

func TestNamedLogger(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)

	named := log.Named("http")
	assert.NotNil(t, named)
	named.Info("named logger works")
	_ = log.Sync()
}
