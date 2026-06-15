# Logger Package Design

**Date:** 2026-06-12
**Status:** Approved

## Overview

A Go package that provides structured logging via `go.uber.org/zap`, configured entirely through a `.env` file. Supports simultaneous console and file output with size+time-based rotation, a global zero-config default instance, a constructor for custom instances, and framework-agnostic middleware sub-packages for Gin, Echo, Fiber, Chi, and standard `net/http`.

---

## Architecture & Package Layout

```
github.com/kingsouphasin/logger/
├── logger.go          # Logger interface + global default instance
├── config.go          # Config struct + .env loading with godotenv
├── builder.go         # New() constructor — builds zap core from config
├── context.go         # FromContext() / WithContext() helpers
├── rotate.go          # lumberjack setup for file rotation
├── go.mod
├── go.sum
└── middleware/
    ├── gin/
    │   └── middleware.go
    ├── echo/
    │   └── middleware.go
    ├── fiber/
    │   └── middleware.go
    ├── chi/
    │   └── middleware.go
    └── http/
        └── middleware.go
```

**Dependencies:**

| Package | Purpose |
|---|---|
| `go.uber.org/zap` | Core structured logger |
| `gopkg.in/natefinch/lumberjack.v2` | File rotation (size-based); time-based rotation handled via a background goroutine that reopens the file on a daily tick |
| `github.com/joho/godotenv` | `.env` file loading |

The core `logger` package imports no framework. Each middleware sub-package imports only its own framework.

---

## Configuration & `.env` Variables

All settings are optional. The package works with zero config using sensible defaults.

```env
# General
LOGGER_ENV=development          # development | production (default: production)
LOGGER_LEVEL=info               # debug | info | warn | error | fatal (default: info)
LOGGER_CALLER=true              # include file:line in logs (default: false)

# Console output
LOGGER_CONSOLE=true             # enable console output (default: true)

# File output
LOGGER_FILE=true                # enable file output (default: true)
LOGGER_FILE_PATH=./logs/app.log # log file path (default: ./logs/app.log)
LOGGER_MAX_SIZE_MB=100          # max file size before rotation (default: 100)
LOGGER_MAX_BACKUPS=30           # max rotated files to keep (default: 30)
LOGGER_MAX_AGE_DAYS=30          # delete rotated files older than N days (default: 30)
LOGGER_COMPRESS=true            # gzip compress rotated files (default: false)
```

**Mode behavior:**
- `LOGGER_ENV=development` → colored, human-readable console output (`zapcore.NewConsoleEncoder`)
- `LOGGER_ENV=production` → structured JSON output (`zapcore.NewJSONEncoder`)

**Config struct:**

```go
type Config struct {
    Env        string // development | production
    Level      string // debug | info | warn | error | fatal
    Caller     bool
    Console    bool
    File       bool
    FilePath   string
    MaxSizeMB  int
    MaxBackups int
    MaxAgeDays int
    Compress   bool
}
```

---

## Logger Interface & API

```go
type Logger interface {
    Debug(msg string, fields ...zap.Field)
    Info(msg string, fields ...zap.Field)
    Warn(msg string, fields ...zap.Field)
    Error(msg string, fields ...zap.Field)
    Fatal(msg string, fields ...zap.Field)

    // Sugared (key-value style)
    Debugw(msg string, keysAndValues ...any)
    Infow(msg string, keysAndValues ...any)
    Warnw(msg string, keysAndValues ...any)
    Errorw(msg string, keysAndValues ...any)

    // Child loggers
    With(fields ...zap.Field) Logger
    Named(name string) Logger

    // Dynamic level change at runtime
    SetLevel(level string) error

    // Flush buffered logs — call on shutdown
    Sync() error
}
```

**Usage examples:**

```go
// Zero-config global instance (auto-loads .env)
import "github.com/kingsouphasin/logger"

logger.Info("server started", zap.String("port", "8080"))
logger.Infow("user login", "user_id", 42, "ip", "1.2.3.4")

// Child logger with persistent fields
authLog := logger.With(zap.String("service", "auth"))
authLog.Info("token issued")

// Named logger
httpLog := logger.Named("http")
httpLog.Warn("slow request", zap.Duration("latency", d))

// Custom instance
cfg := logger.Config{Env: "production", Level: "debug", File: true}
log, err := logger.New(cfg)
log.Info("custom instance ready")

// From context (any framework)
log := logger.FromContext(ctx)
log.Info("handling request")
```

**Middleware usage:**

```go
// Gin
import ginlogger "github.com/kingsouphasin/logger/middleware/gin"
r.Use(ginlogger.Middleware())

// Echo
import echologger "github.com/kingsouphasin/logger/middleware/echo"
e.Use(echologger.Middleware())

// net/http
import httplogger "github.com/kingsouphasin/logger/middleware/http"
http.Handle("/", httplogger.Middleware()(myHandler))
```

Each middleware logs: method, path, status code, latency — and injects the logger into `ctx`.

---

## Error Handling

- `New()` returns `(Logger, error)` — callers handle bad config explicitly.
- The global default instance **never panics** — falls back to defaults and logs a warning to stderr if `.env` is missing or invalid.
- `Fatal` calls `os.Exit(1)` after logging (standard zap behavior).
- `defer logger.Sync()` should be called on shutdown to flush buffered logs.

**Error scenarios:**

| Situation | Behavior |
|---|---|
| `.env` file not found | Use defaults, no error |
| Invalid `LOGGER_LEVEL` value | Fall back to `info`, log warning to stderr |
| Log directory doesn't exist | Auto-create with `os.MkdirAll` |
| File write permission denied | Return error from `New()`, global falls back to console-only |
| Invalid numeric env value | Fall back to default value |

---

## Testing

- The `Logger` interface makes consumer code trivially mockable.
- Core package tests cover: config parsing edge cases, output routing (console/file/both), level filtering, child logger field inheritance.
- Each middleware tested with its framework's test utilities (`httptest` for net/http, Gin test mode, etc.).
- File output tests use an in-memory writer — no real files created in tests.
