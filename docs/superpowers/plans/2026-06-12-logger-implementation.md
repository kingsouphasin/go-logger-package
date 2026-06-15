# Logger Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready Go logger package wrapping zap with `.env` configuration, dual console+file output, size+time-based rotation, and framework-agnostic middleware sub-packages for net/http, Gin, Echo, Fiber, and Chi.

**Architecture:** Single root module (`github.com/kingsouphasin/logger`) for the core — zero framework imports. Each framework middleware lives in its own Go module under `middleware/<framework>/` so users who only need the core don't pull in framework transitive deps. A `go.work` workspace links all modules for local development.

**Tech Stack:** `go.uber.org/zap`, `gopkg.in/natefinch/lumberjack.v2`, `github.com/joho/godotenv`, `github.com/stretchr/testify`, plus per-middleware framework packages.

---

## File Map

```
github.com/kingsouphasin/logger/
├── go.mod                         # root module
├── go.work                        # workspace for local dev
├── logger.go                      # Logger interface + package-level functions + init()
├── config.go                      # Config struct + LoadConfig() + defaults
├── builder.go                     # New() + zapLogger struct + all zapLogger methods
├── context.go                     # WithContext() + FromContext()
├── rotate.go                      # newRotatingWriter() + startTimeRotation()
├── config_test.go
├── builder_test.go
├── context_test.go
├── logger_test.go
└── middleware/
    ├── http/
    │   ├── go.mod                 # module github.com/kingsouphasin/logger/middleware/http
    │   ├── middleware.go          # package httplogger
    │   └── middleware_test.go
    ├── gin/
    │   ├── go.mod                 # module github.com/kingsouphasin/logger/middleware/gin
    │   ├── middleware.go          # package ginlogger
    │   └── middleware_test.go
    ├── echo/
    │   ├── go.mod                 # module github.com/kingsouphasin/logger/middleware/echo
    │   ├── middleware.go          # package echologger
    │   └── middleware_test.go
    ├── fiber/
    │   ├── go.mod                 # module github.com/kingsouphasin/logger/middleware/fiber
    │   ├── middleware.go          # package fiberlogger
    │   └── middleware_test.go
    └── chi/
        ├── go.mod                 # module github.com/kingsouphasin/logger/middleware/chi
        ├── middleware.go          # package chilogger
        └── middleware_test.go
```

---

## Task 1: Project Setup

**Files:**
- Create: `go.mod`
- Create: `go.work`
- Create: `middleware/http/go.mod`
- Create: `middleware/gin/go.mod`
- Create: `middleware/echo/go.mod`
- Create: `middleware/fiber/go.mod`
- Create: `middleware/chi/go.mod`

- [ ] **Step 1: Initialize root module**

```bash
cd /Users/mac/Desktop/Logger-package
go mod init github.com/kingsouphasin/logger
go get go.uber.org/zap@latest
go get gopkg.in/natefinch/lumberjack.v2@latest
go get github.com/joho/godotenv@latest
go get github.com/stretchr/testify@latest
```

- [ ] **Step 2: Initialize middleware modules**

```bash
mkdir -p middleware/http middleware/gin middleware/echo middleware/fiber middleware/chi

cd middleware/http
go mod init github.com/kingsouphasin/logger/middleware/http
go get github.com/kingsouphasin/logger@latest
go get go.uber.org/zap@latest
cd ../..

cd middleware/gin
go mod init github.com/kingsouphasin/logger/middleware/gin
go get github.com/gin-gonic/gin@latest
go get github.com/kingsouphasin/logger@latest
go get go.uber.org/zap@latest
cd ../..

cd middleware/echo
go mod init github.com/kingsouphasin/logger/middleware/echo
go get github.com/labstack/echo/v4@latest
go get github.com/kingsouphasin/logger@latest
go get go.uber.org/zap@latest
cd ../..

cd middleware/fiber
go mod init github.com/kingsouphasin/logger/middleware/fiber
go get github.com/gofiber/fiber/v2@latest
go get github.com/kingsouphasin/logger@latest
go get go.uber.org/zap@latest
cd ../..

cd middleware/chi
go mod init github.com/kingsouphasin/logger/middleware/chi
go get github.com/go-chi/chi/v5@latest
go get github.com/kingsouphasin/logger@latest
go get go.uber.org/zap@latest
cd ../..
```

- [ ] **Step 3: Create go.work workspace**

```bash
cd /Users/mac/Desktop/Logger-package
go work init .
go work use ./middleware/http ./middleware/gin ./middleware/echo ./middleware/fiber ./middleware/chi
```

Expected `go.work`:
```
go 1.21

use (
    .
    ./middleware/chi
    ./middleware/echo
    ./middleware/fiber
    ./middleware/gin
    ./middleware/http
)
```

- [ ] **Step 4: Add testify to each middleware module**

```bash
cd middleware/http  && go get github.com/stretchr/testify@latest && cd ../..
cd middleware/gin   && go get github.com/stretchr/testify@latest && cd ../..
cd middleware/echo  && go get github.com/stretchr/testify@latest && cd ../..
cd middleware/fiber && go get github.com/stretchr/testify@latest && cd ../..
cd middleware/chi   && go get github.com/stretchr/testify@latest && cd ../..
```

- [ ] **Step 5: Commit**

```bash
git init
git add go.mod go.work middleware/http/go.mod middleware/gin/go.mod middleware/echo/go.mod middleware/fiber/go.mod middleware/chi/go.mod
git commit -m "chore: initialize multi-module workspace"
```

---

## Task 2: Config

**Files:**
- Create: `config.go`
- Create: `config_test.go`

- [ ] **Step 1: Write failing tests**

Create `config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package
go test ./... -run "TestDefaultConfig|TestConfigFromEnv|TestInvalidInt|TestInvalidBool" -v 2>&1 | head -20
```

Expected: `FAIL` — `LoadConfig undefined`

- [ ] **Step 3: Implement config.go**

Create `config.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -run "TestDefaultConfig|TestConfigFromEnv|TestInvalidInt|TestInvalidBool" -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add Config struct and .env loading"
```

---

## Task 3: Log Rotation Helper

**Files:**
- Create: `rotate.go`

- [ ] **Step 1: Create rotate.go**

```go
package logger

import (
	"context"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func newRotatingWriter(cfg Config) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

func startTimeRotation(ctx context.Context, lj *lumberjack.Logger) {
	go func() {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		timer := time.NewTimer(time.Until(next))
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				_ = lj.Rotate()
				next = next.Add(24 * time.Hour)
				timer.Reset(time.Until(next))
			}
		}
	}()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add rotate.go
git commit -m "feat: add lumberjack rotation helper with daily time rotation"
```

---

## Task 4: Builder and zapLogger

**Files:**
- Create: `builder.go`
- Create: `builder_test.go`

- [ ] **Step 1: Write failing tests**

Create `builder_test.go`:

```go
package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./... -run "TestNew|TestSetLevel|TestWith|TestNamed" -v 2>&1 | head -20
```

Expected: `FAIL` — `New undefined`

- [ ] **Step 3: Implement builder.go**

Create `builder.go`:

```go
package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type zapLogger struct {
	z      *zap.Logger
	sugar  *zap.SugaredLogger
	level  zap.AtomicLevel
	cancel context.CancelFunc
}

func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: invalid level %q, falling back to info\n", cfg.Level)
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	enc := buildEncoder(cfg)
	var cores []zapcore.Core

	if cfg.Console {
		cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), level))
	}

	var cancel context.CancelFunc
	if cfg.File {
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return nil, fmt.Errorf("logger: create log directory: %w", err)
		}
		lj := newRotatingWriter(cfg)
		cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(lj), level))

		ctx, c := context.WithCancel(context.Background())
		cancel = c
		startTimeRotation(ctx, lj)
	}

	opts := []zap.Option{zap.WithCaller(cfg.Caller)}
	z := zap.New(zapcore.NewTee(cores...), opts...)

	return &zapLogger{
		z:      z,
		sugar:  z.Sugar(),
		level:  level,
		cancel: cancel,
	}, nil
}

func (l *zapLogger) Debug(msg string, fields ...zap.Field) { l.z.Debug(msg, fields...) }
func (l *zapLogger) Info(msg string, fields ...zap.Field)  { l.z.Info(msg, fields...) }
func (l *zapLogger) Warn(msg string, fields ...zap.Field)  { l.z.Warn(msg, fields...) }
func (l *zapLogger) Error(msg string, fields ...zap.Field) { l.z.Error(msg, fields...) }
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) { l.z.Fatal(msg, fields...) }

func (l *zapLogger) Debugw(msg string, kv ...any) { l.sugar.Debugw(msg, kv...) }
func (l *zapLogger) Infow(msg string, kv ...any)  { l.sugar.Infow(msg, kv...) }
func (l *zapLogger) Warnw(msg string, kv ...any)  { l.sugar.Warnw(msg, kv...) }
func (l *zapLogger) Errorw(msg string, kv ...any) { l.sugar.Errorw(msg, kv...) }

func (l *zapLogger) With(fields ...zap.Field) Logger {
	z := l.z.With(fields...)
	return &zapLogger{z: z, sugar: z.Sugar(), level: l.level}
}

func (l *zapLogger) Named(name string) Logger {
	z := l.z.Named(name)
	return &zapLogger{z: z, sugar: z.Sugar(), level: l.level}
}

func (l *zapLogger) SetLevel(level string) error {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return err
	}
	l.level.SetLevel(lvl)
	return nil
}

func (l *zapLogger) Sync() error {
	if l.cancel != nil {
		l.cancel()
	}
	return l.z.Sync()
}

func parseLevel(s string) (zap.AtomicLevel, error) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return zap.NewAtomicLevel(), err
	}
	return zap.NewAtomicLevelAt(l), nil
}

func buildEncoder(cfg Config) zapcore.Encoder {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	if cfg.Env == "development" {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(encCfg)
	}
	return zapcore.NewJSONEncoder(encCfg)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -run "TestNew|TestSetLevel|TestWith|TestNamed" -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
git add builder.go builder_test.go
git commit -m "feat: add New() constructor and zapLogger implementation"
```

---

## Task 5: Logger Interface and Global Instance

**Files:**
- Create: `logger.go`
- Create: `logger_test.go`

- [ ] **Step 1: Write failing tests**

Create `logger_test.go`:

```go
package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGlobalFunctionsDoNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { Debug("debug msg") })
	assert.NotPanics(t, func() { Info("info msg") })
	assert.NotPanics(t, func() { Warn("warn msg") })
	assert.NotPanics(t, func() { Error("error msg") })
	assert.NotPanics(t, func() { Debugw("debug sugared", "key", "val") })
	assert.NotPanics(t, func() { Infow("info sugared", "key", "val") })
	assert.NotPanics(t, func() { Warnw("warn sugared", "key", "val") })
	assert.NotPanics(t, func() { Errorw("error sugared", "key", "val") })
}

func TestGlobalWithReturnsLogger(t *testing.T) {
	child := With(zap.String("service", "test"))
	assert.NotNil(t, child)
}

func TestGlobalNamedReturnsLogger(t *testing.T) {
	named := Named("component")
	assert.NotNil(t, named)
}

func TestGlobalSetLevel(t *testing.T) {
	require.NoError(t, SetLevel("debug"))
	require.NoError(t, SetLevel("info"))
	assert.Error(t, SetLevel("badlevel"))
}

func TestSetDefault(t *testing.T) {
	cfg := Config{Env: "production", Level: "warn", Console: true, File: false}
	custom, err := New(cfg)
	require.NoError(t, err)

	SetDefault(custom)
	assert.NotPanics(t, func() { Info("via custom default") })
	_ = Sync()
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./... -run "TestGlobal|TestSetDefault" -v 2>&1 | head -20
```

Expected: `FAIL` — `Info undefined` (or similar)

- [ ] **Step 3: Implement logger.go**

Create `logger.go`:

```go
package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

// Logger is the interface implemented by this package's logger and all child loggers.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)

	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)

	With(fields ...zap.Field) Logger
	Named(name string) Logger
	SetLevel(level string) error
	Sync() error
}

// Re-export common zap field constructors so callers don't need to import zap directly.
var (
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Duration = zap.Duration
	Any      = zap.Any
	Err      = zap.Error
)

var defaultLogger Logger

func init() {
	cfg := LoadConfig()
	l, err := New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: file output failed (%v), falling back to console\n", err)
		fallback := cfg
		fallback.File = false
		l, _ = New(fallback)
	}
	defaultLogger = l
}

// SetDefault replaces the global default logger. Useful in tests and for custom bootstrap.
func SetDefault(l Logger) { defaultLogger = l }

func Debug(msg string, fields ...zap.Field)  { defaultLogger.Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)   { defaultLogger.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)   { defaultLogger.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field)  { defaultLogger.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field)  { defaultLogger.Fatal(msg, fields...) }
func Debugw(msg string, kv ...any)           { defaultLogger.Debugw(msg, kv...) }
func Infow(msg string, kv ...any)            { defaultLogger.Infow(msg, kv...) }
func Warnw(msg string, kv ...any)            { defaultLogger.Warnw(msg, kv...) }
func Errorw(msg string, kv ...any)           { defaultLogger.Errorw(msg, kv...) }
func With(fields ...zap.Field) Logger        { return defaultLogger.With(fields...) }
func Named(name string) Logger               { return defaultLogger.Named(name) }
func SetLevel(level string) error            { return defaultLogger.SetLevel(level) }
func Sync() error                            { return defaultLogger.Sync() }
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -run "TestGlobal|TestSetDefault" -v
```

Expected: All `PASS`

- [ ] **Step 5: Run all core tests**

```bash
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 6: Commit**

```bash
git add logger.go logger_test.go
git commit -m "feat: add Logger interface, global instance, and zap field re-exports"
```

---

## Task 6: Context Helpers

**Files:**
- Create: `context.go`
- Create: `context_test.go`

- [ ] **Step 1: Write failing tests**

Create `context_test.go`:

```go
package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithContextAndFromContext(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)

	ctx := WithContext(context.Background(), log)
	retrieved := FromContext(ctx)
	assert.Equal(t, log, retrieved)
	_ = log.Sync()
}

func TestFromContextReturnsDefaultWhenNotSet(t *testing.T) {
	ctx := context.Background()
	log := FromContext(ctx)
	assert.NotNil(t, log)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./... -run "TestWithContext|TestFromContext" -v 2>&1 | head -20
```

Expected: `FAIL` — `WithContext undefined`

- [ ] **Step 3: Implement context.go**

Create `context.go`:

```go
package logger

import "context"

type contextKey struct{}

func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(contextKey{}).(Logger); ok {
		return l
	}
	return defaultLogger
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
git add context.go context_test.go
git commit -m "feat: add WithContext and FromContext helpers"
```

---

## Task 7: net/http Middleware

**Files:**
- Create: `middleware/http/middleware.go`
- Create: `middleware/http/middleware_test.go`

- [ ] **Step 1: Write failing tests**

Create `middleware/http/middleware_test.go`:

```go
package httplogger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareInjectsLoggerIntoContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		assert.NotNil(t, log)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	Middleware()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddlewareCapturesStatusCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	Middleware()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMiddlewareDefaultsTo200(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no explicit WriteHeader
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()
	Middleware()(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/http
go test ./... -v 2>&1 | head -20
```

Expected: `FAIL` — `Middleware undefined`

- [ ] **Step 3: Implement middleware.go**

Create `middleware/http/middleware.go`:

```go
package httplogger

import (
	"net/http"
	"time"

	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			log := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			ctx := logger.WithContext(r.Context(), log)

			next.ServeHTTP(rw, r.WithContext(ctx))

			log.Info("request completed",
				zap.Int("status", rw.statusCode),
				zap.Duration("latency", time.Since(start)),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/http
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
cd /Users/mac/Desktop/Logger-package
git add middleware/http/
git commit -m "feat: add net/http logger middleware"
```

---

## Task 8: Gin Middleware

**Files:**
- Create: `middleware/gin/middleware.go`
- Create: `middleware/gin/middleware_test.go`

- [ ] **Step 1: Write failing tests**

Create `middleware/gin/middleware_test.go`:

```go
package ginlogger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
)

func TestGinMiddlewareInjectsLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/test", func(c *gin.Context) {
		log := logger.FromContext(c.Request.Context())
		assert.NotNil(t, log)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGinMiddlewareCaptures404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/exists", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/gin
go test ./... -v 2>&1 | head -20
```

Expected: `FAIL` — `Middleware undefined`

- [ ] **Step 3: Implement middleware.go**

Create `middleware/gin/middleware.go`:

```go
package ginlogger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		log := logger.With(
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
		)
		ctx := logger.WithContext(c.Request.Context(), log)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		log.Info("request completed",
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/gin
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
cd /Users/mac/Desktop/Logger-package
git add middleware/gin/
git commit -m "feat: add Gin logger middleware"
```

---

## Task 9: Echo Middleware

**Files:**
- Create: `middleware/echo/middleware.go`
- Create: `middleware/echo/middleware_test.go`

- [ ] **Step 1: Write failing tests**

Create `middleware/echo/middleware_test.go`:

```go
package echologger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kingsouphasin/logger"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEchoMiddlewareInjectsLogger(t *testing.T) {
	e := echo.New()
	e.Use(Middleware())
	e.GET("/test", func(c echo.Context) error {
		log := logger.FromContext(c.Request().Context())
		assert.NotNil(t, log)
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestEchoMiddlewareCaptures500(t *testing.T) {
	e := echo.New()
	e.Use(Middleware())
	e.GET("/error", func(c echo.Context) error {
		return c.NoContent(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/echo
go test ./... -v 2>&1 | head -20
```

Expected: `FAIL` — `Middleware undefined`

- [ ] **Step 3: Implement middleware.go**

Create `middleware/echo/middleware.go`:

```go
package echologger

import (
	"time"

	"github.com/kingsouphasin/logger"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			log := logger.With(
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
			)
			ctx := logger.WithContext(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			log.Info("request completed",
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", time.Since(start)),
			)
			return err
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/echo
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
cd /Users/mac/Desktop/Logger-package
git add middleware/echo/
git commit -m "feat: add Echo logger middleware"
```

---

## Task 10: Fiber Middleware

**Files:**
- Create: `middleware/fiber/middleware.go`
- Create: `middleware/fiber/middleware_test.go`

> Note: Fiber uses its own `*fasthttp.RequestCtx` (not `context.Context`), so logger injection uses `c.Locals` instead of `context.WithValue`. A `FromFiberCtx` helper is provided for consumers.

- [ ] **Step 1: Write failing tests**

Create `middleware/fiber/middleware_test.go`:

```go
package fiberlogger

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiberMiddlewareInjectsLogger(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		log := FromFiberCtx(c)
		assert.NotNil(t, log)
		return c.SendStatus(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFiberMiddlewareFromFiberCtxFallsBackToDefault(t *testing.T) {
	app := fiber.New()
	app.Get("/no-middleware", func(c *fiber.Ctx) error {
		log := FromFiberCtx(c)
		assert.NotNil(t, log)
		assert.Equal(t, logger.FromContext(c.Context()), log)
		return c.SendStatus(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/no-middleware", nil)
	require.NoError(t, err)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/fiber
go test ./... -v 2>&1 | head -20
```

Expected: `FAIL` — `Middleware undefined`

- [ ] **Step 3: Implement middleware.go**

Create `middleware/fiber/middleware.go`:

```go
package fiberlogger

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

const loggerKey = "logger"

func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		log := logger.With(
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)
		c.Locals(loggerKey, log)

		err := c.Next()

		log.Info("request completed",
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
		)
		return err
	}
}

// FromFiberCtx retrieves the logger injected by Middleware from Fiber locals.
// Falls back to the logger stored in the request's context (or the global default).
func FromFiberCtx(c *fiber.Ctx) logger.Logger {
	if l, ok := c.Locals(loggerKey).(logger.Logger); ok {
		return l
	}
	return logger.FromContext(c.Context())
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/fiber
go test ./... -v
```

Expected: All `PASS`

- [ ] **Step 5: Commit**

```bash
cd /Users/mac/Desktop/Logger-package
git add middleware/fiber/
git commit -m "feat: add Fiber logger middleware with FromFiberCtx helper"
```

---

## Task 11: Chi Middleware

**Files:**
- Create: `middleware/chi/middleware.go`
- Create: `middleware/chi/middleware_test.go`

- [ ] **Step 1: Write failing tests**

Create `middleware/chi/middleware_test.go`:

```go
package chilogger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
)

func TestChiMiddlewareInjectsLogger(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Middleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		assert.NotNil(t, log)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChiMiddlewareCapturesStatus(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Middleware)
	r.Get("/created", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/created", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/chi
go test ./... -v 2>&1 | head -20
```

Expected: `FAIL` — `Middleware undefined`

- [ ] **Step 3: Implement middleware.go**

Create `middleware/chi/middleware.go`:

```go
package chilogger

import (
	"net/http"
	"time"

	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		log := logger.With(
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		ctx := logger.WithContext(r.Context(), log)

		next.ServeHTTP(rw, r.WithContext(ctx))

		log.Info("request completed",
			zap.Int("status", rw.statusCode),
			zap.Duration("latency", time.Since(start)),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
```

- [ ] **Step 4: Run all tests across all modules**

```bash
cd /Users/mac/Desktop/Logger-package/middleware/chi
go test ./... -v

# Then run all modules from root
cd /Users/mac/Desktop/Logger-package
go test ./...
cd middleware/http  && go test ./... && cd ../..
cd middleware/gin   && go test ./... && cd ../..
cd middleware/echo  && go test ./... && cd ../..
cd middleware/fiber && go test ./... && cd ../..
cd middleware/chi   && go test ./... && cd ../..
```

Expected: All `PASS` across all modules

- [ ] **Step 5: Final commit**

```bash
cd /Users/mac/Desktop/Logger-package
git add middleware/chi/
git commit -m "feat: add Chi logger middleware"
git tag v0.1.0
```
