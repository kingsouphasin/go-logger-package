# logger

A production-ready Go structured logger built on [zap](https://github.com/uber-go/zap), with dual console + file output, automatic log rotation, and drop-in middleware for the most common Go HTTP frameworks.

```
github.com/kingsouphasin/logger
```

---

## Features

- **Zero-config** — works out of the box with sensible defaults
- **Dual output** — writes to console and a rotating file simultaneously
- **Log rotation** — size-based and daily time-based (whichever comes first)
- **Environment-driven config** — all settings via `.env` or environment variables
- **Development & production modes** — colored console output in dev, JSON in prod
- **Context propagation** — carry a logger through `context.Context`
- **Framework middleware** — net/http, Gin, Echo, Fiber, Chi (each an independent module)
- **Request enrichment** — auto-logs `request_id`, IP, user-agent, sizes, latency
- **File upload safety** — logs multipart file metadata (name, size) but never file content
- **Sensitive query redaction** — tokens, API keys, passwords are automatically masked

---

## Installation

```bash
go get github.com/kingsouphasin/logger
```

Each middleware lives in its own module so you only pull in the framework deps you actually use:

```bash
# net/http
go get github.com/kingsouphasin/logger/middleware/http

# Gin
go get github.com/kingsouphasin/logger/middleware/gin

# Echo
go get github.com/kingsouphasin/logger/middleware/echo

# Fiber
go get github.com/kingsouphasin/logger/middleware/fiber

# Chi
go get github.com/kingsouphasin/logger/middleware/chi
```

---

## Two Ways to Log

This package has two distinct logging modes. Understanding the difference is the key to using it correctly.

### 1. Global logger — for app-level events

```go
logger.Info("server started", logger.String("port", "8080"))
```

- Available everywhere, zero setup
- **No `request_id`** — there is no HTTP request at this point, so there is nothing to identify
- Use this for startup, shutdown, background jobs, cron tasks, and anything outside an HTTP request

### 2. Context logger — for request-scoped events

```go
log := logger.FromContext(ctx)  // inside a handler or service function
log.Info("processing order")    // automatically includes request_id and all request fields
```

- Retrieved from `context.Context` inside an HTTP handler
- **Has `request_id`** — injected by the middleware before your handler runs
- Use this for everything that happens during an HTTP request

> **Rule of thumb:** use `logger.Info(...)` for app events, use `logger.FromContext(ctx)` inside handlers and service functions called from handlers.

---

## Quick Start

### Zero-config (global logger)

No setup required. Import the package and start logging:

```go
package main

import "github.com/kingsouphasin/logger"

func main() {
    defer logger.Sync()

    // These use the global logger — no request_id, and that is correct.
    // There is no HTTP request here, so there is nothing to identify.
    logger.Info("server started", logger.String("port", "8080"))
    logger.Warn("high memory usage", logger.Int("mb", 512))
    logger.Error("database error", logger.Err(err))
}
```

The global logger is initialized automatically on import using `LoadConfig()`, which reads from `.env` and environment variables.

### Sugared (key-value) style

```go
logger.Infow("user signed in", "user_id", 42, "email", "user@example.com")
logger.Errorw("payment failed", "order_id", "ORD-001", "reason", "insufficient funds")
```

---

## Configuration

Create a `.env` file in your project root. All fields are optional — defaults are shown:

```env
# "development" shows colored console output
# "production"  outputs JSON (default)
LOGGER_ENV=production

# Minimum log level: debug | info | warn | error | fatal
LOGGER_LEVEL=info

# Include caller file:line in every log entry
LOGGER_CALLER=false

# Write to console (stdout)
LOGGER_CONSOLE=true

# Write to a rotating log file
LOGGER_FILE=true
LOGGER_FILE_PATH=./logs/app.log

# Rotate when the file reaches this size (megabytes)
LOGGER_MAX_SIZE_MB=100

# Number of rotated files to keep
LOGGER_MAX_BACKUPS=30

# Delete rotated files older than this many days
LOGGER_MAX_AGE_DAYS=30

# Compress rotated files with gzip
LOGGER_COMPRESS=false
```

### Log rotation

Rotation is triggered by whichever comes first:
- The file reaches `LOGGER_MAX_SIZE_MB`
- Midnight (daily rotation)

Old files are automatically deleted based on `LOGGER_MAX_BACKUPS` and `LOGGER_MAX_AGE_DAYS`.

---

## Custom Instance

Use `New()` when you need a logger with specific settings (e.g. for a sub-service or test):

```go
package main

import (
    "log"
    "github.com/kingsouphasin/logger"
)

func main() {
    cfg := logger.Config{
        Env:        "development",
        Level:      "debug",
        Console:    true,
        File:       true,
        FilePath:   "./logs/myapp.log",
        MaxSizeMB:  50,
        MaxBackups: 7,
        MaxAgeDays: 7,
        Compress:   true,
        Caller:     true,
    }

    log, err := logger.New(cfg)
    if err != nil {
        log.Fatal("failed to init logger")
    }
    defer log.Sync()

    log.Info("custom logger ready")

    // Optionally replace the global default
    logger.SetDefault(log)
}
```

---

## Structured Fields

The package re-exports common zap field helpers so you don't need to import zap directly:

```go
logger.Info("order placed",
    logger.String("order_id", "ORD-123"),
    logger.Int("items", 3),
    logger.Float64("total", 99.95),
    logger.Bool("paid", true),
    logger.Duration("processing", time.Second),
    logger.Any("metadata", map[string]string{"source": "web"}),
    logger.Err(err),
)
```

### Child loggers

`With()` creates a child logger that includes a fixed set of fields in every message:

```go
userLog := logger.With(logger.String("user_id", "u-42"))
userLog.Info("profile updated")   // includes user_id
userLog.Warn("invalid input")     // includes user_id
```

`Named()` adds a component name prefix:

```go
dbLog := logger.Named("database")
dbLog.Info("connected")           // {"logger":"database","msg":"connected",...}
```

### Dynamic log level

```go
logger.SetLevel("debug")   // enable verbose logging at runtime
logger.SetLevel("warn")    // quiet things down
```

---

## Context Integration

Carry a logger through `context.Context` to pass request-scoped fields (like `request_id`) deep into your service code without threading the logger as a parameter.

```go
// Store a logger in context
ctx := logger.WithContext(r.Context(), log)

// Retrieve it anywhere downstream
log := logger.FromContext(ctx)
log.Info("processing payment")    // inherits all fields from the stored logger
```

`FromContext` falls back to the global default logger if none is stored.

---

## Middleware

### How `request_id` flows through your application

When a request arrives, the middleware runs first and sets everything up before your handler is called:

```
HTTP Request
     │
     ▼
┌─────────────────────────────────────────────────────┐
│  Middleware                                         │
│  1. Read X-Request-ID header                        │
│     └─ if missing → generate UUID v4               │
│  2. Set X-Request-ID on the response header         │
│  3. Create a child logger with request_id attached  │
│  4. Store that logger in context.Context            │
└─────────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────┐
│  Your Handler                                       │
│                                                     │
│  log := logger.FromContext(ctx)  ← has request_id  │
│  log.Info("doing work")          ← has request_id  │
│                                                     │
│  processOrder(ctx, id)           ← pass ctx down   │
│    └─ log := logger.FromContext(ctx)                │
│       log.Info("order done")     ← still has it    │
└─────────────────────────────────────────────────────┘
     │
     ▼
Middleware logs "request completed" with status + latency
```

**Key rule:** always pass `ctx` down through your service functions and always use `logger.FromContext(ctx)` inside them — never call `logger.Info(...)` directly inside a handler, or you will lose the `request_id`.

```go
// ✅ correct — request_id flows through
func placeOrder(ctx context.Context) {
    log := logger.FromContext(ctx)
    log.Info("order placed")   // {"request_id":"abc", "msg":"order placed"}
}

// ❌ wrong — request_id is lost
func placeOrder(ctx context.Context) {
    logger.Info("order placed")   // {"msg":"order placed"} — no request_id
}
```

### Using `Handle` — skip `FromContext` entirely

Each middleware package provides a `Handle` wrapper that injects the logger directly into your handler as a parameter. You never need to call `FromContext` yourself:

```go
// Without Handle — you call FromContext manually
mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    log := logger.FromContext(r.Context())
    log.Info("hello")
})

// With Handle — logger is injected automatically
mux.HandleFunc("/hello", httplogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
    log.Info("hello")  // log is ready, no FromContext needed
}))
```

The `Handle` wrapper works for all frameworks — see each framework section below for its specific signature.

All middleware packages:
- Generate or propagate `X-Request-ID` (read from request header, or generate UUID v4)
- Set `X-Request-ID` on the response header
- Store a request-scoped child logger in context (accessible via `logger.FromContext`)
- Log a `request completed` entry with: `status`, `latency`, `response_size`
- Log request details: `request_id`, `method`, `path`, `query`, `ip`, `user_agent`, `content_type`, `request_size`
- Redact sensitive query parameters automatically (`token`, `api_key`, `password`, etc.)
- For `multipart/form-data`: log `uploaded_files` with filename/size/content-type — never file content

### net/http

```go
package main

import (
    "net/http"

    "github.com/kingsouphasin/logger"
    httplogger "github.com/kingsouphasin/logger/middleware/http"
)

func main() {
    mux := http.NewServeMux()

    // Option A: use Handle — logger injected as parameter
    mux.HandleFunc("/hello", httplogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        log.Info("handling request")
        w.Write([]byte("OK"))
    }))

    // Option B: use FromContext manually
    mux.HandleFunc("/hello2", func(w http.ResponseWriter, r *http.Request) {
        log := logger.FromContext(r.Context())
        log.Info("handling request")
        w.Write([]byte("OK"))
    })

    http.ListenAndServe(":8080", httplogger.Middleware()(mux))
}
```

### Gin

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/kingsouphasin/logger"
    ginlogger "github.com/kingsouphasin/logger/middleware/gin"
)

func main() {
    r := gin.New()
    r.Use(ginlogger.Middleware())

    // Option A: use Handle — logger injected as second parameter
    r.GET("/users/:id", ginlogger.Handle(func(c *gin.Context, log logger.Logger) {
        log.Info("fetching user", logger.String("id", c.Param("id")))
        c.JSON(200, gin.H{"id": c.Param("id")})
    }))

    // Option B: use FromContext manually
    r.GET("/users2/:id", func(c *gin.Context) {
        log := logger.FromContext(c.Request.Context())
        log.Info("fetching user", logger.String("id", c.Param("id")))
        c.JSON(200, gin.H{"id": c.Param("id")})
    })

    r.Run(":8080")
}
```

### Echo

```go
package main

import (
    "net/http"

    "github.com/kingsouphasin/logger"
    echologger "github.com/kingsouphasin/logger/middleware/echo"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()
    e.Use(echologger.Middleware())

    // Option A: use Handle — logger injected as second parameter
    e.GET("/users/:id", echologger.Handle(func(c echo.Context, log logger.Logger) error {
        log.Info("fetching user", logger.String("id", c.Param("id")))
        return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
    }))

    // Option B: use FromContext manually
    e.GET("/users2/:id", func(c echo.Context) error {
        log := logger.FromContext(c.Request().Context())
        log.Info("fetching user", logger.String("id", c.Param("id")))
        return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
    })

    e.Start(":8080")
}
```

### Fiber

Fiber uses [fasthttp](https://github.com/valyala/fasthttp) which is not compatible with standard `context.Context`. Use `Handle` or `FromFiberCtx` instead of `logger.FromContext`:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/kingsouphasin/logger"
    fiberlogger "github.com/kingsouphasin/logger/middleware/fiber"
)

func main() {
    app := fiber.New()
    app.Use(fiberlogger.Middleware())

    // Option A: use Handle — logger injected as second parameter
    app.Get("/users/:id", fiberlogger.Handle(func(c *fiber.Ctx, log logger.Logger) error {
        log.Info("fetching user", logger.String("id", c.Params("id")))
        return c.JSON(fiber.Map{"id": c.Params("id")})
    }))

    // Option B: use FromFiberCtx manually
    app.Get("/users2/:id", func(c *fiber.Ctx) error {
        log := fiberlogger.FromFiberCtx(c)
        log.Info("fetching user", logger.String("id", c.Params("id")))
        return c.JSON(fiber.Map{"id": c.Params("id")})
    })

    app.Listen(":8080")
}
```

### Chi

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/kingsouphasin/logger"
    chilogger "github.com/kingsouphasin/logger/middleware/chi"
)

func main() {
    r := chi.NewRouter()
    r.Use(chilogger.Middleware)

    // Option A: use Handle — logger injected as third parameter
    r.Get("/users/{id}", chilogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        log.Info("fetching user", logger.String("id", chi.URLParam(r, "id")))
        w.Write([]byte("OK"))
    }))

    // Option B: use FromContext manually
    r.Get("/users2/{id}", func(w http.ResponseWriter, r *http.Request) {
        log := logger.FromContext(r.Context())
        log.Info("fetching user", logger.String("id", chi.URLParam(r, "id")))
        w.Write([]byte("OK"))
    })

    http.ListenAndServe(":8080", r)
}
```

---

## Log Output Examples

### Development mode (`LOGGER_ENV=development`)

Colored, human-readable console output:

```
2024-01-15T10:30:00.000Z  INFO  request completed  {"request_id": "a1b2c3d4", "method": "POST", "path": "/upload", "status": 200, "latency": "12ms"}
```

### Production mode (`LOGGER_ENV=production`)

JSON — one object per line, ready for log aggregators (Datadog, Loki, CloudWatch):

```json
{"level":"info","ts":"2024-01-15T10:30:00.000Z","msg":"request completed","request_id":"a1b2c3d4","method":"GET","path":"/users/42","query":"page=1","ip":"203.0.113.5","user_agent":"Mozilla/5.0","content_type":"application/json","request_size":0,"status":200,"response_size":128,"latency":"3.2ms"}
```

### File upload request

When a client uploads a file via `multipart/form-data`, file metadata is logged — content is never logged:

```json
{
  "level": "info",
  "msg": "request completed",
  "request_id": "f9e8d7c6",
  "method": "POST",
  "path": "/upload",
  "content_type": "multipart/form-data; boundary=...",
  "uploaded_files": [
    {"name": "photo.jpg", "size": 204800, "content_type": "image/jpeg"},
    {"name": "doc.pdf",   "size": 512000, "content_type": "application/pdf"}
  ],
  "status": 200,
  "latency": "45ms"
}
```

### Sensitive query parameter redaction

Parameters like `token`, `api_key`, `password`, `code`, `secret`, `authorization`, `access_token`, `key`, `state`, and `client_secret` are automatically masked:

```
# Request URL: /search?q=golang&api_key=my-secret-key&page=2
# Logged query:
"query": "api_key=%5Bredacted%5D&page=2&q=golang"
```

---

## Tracing a request end-to-end with `request_id`

Every log line emitted via `logger.FromContext(ctx)` automatically carries `request_id`. This lets you filter all logs for a single request in any log aggregator (Datadog, Loki, CloudWatch, etc.) by searching for one value.

```go
// Handler — middleware has already injected request_id into ctx
func orderHandler(w http.ResponseWriter, r *http.Request) {
    log := logger.FromContext(r.Context())
    log.Info("order request received")

    if err := processOrder(r.Context(), "ORD-001"); err != nil {
        log.Error("order failed", logger.Err(err))
        http.Error(w, "error", 500)
        return
    }
    w.WriteHeader(201)
}

// Service layer — still has request_id because we passed ctx
func processOrder(ctx context.Context, orderID string) error {
    log := logger.FromContext(ctx)
    log.Info("processing order", logger.String("order_id", orderID))

    if err := chargeCard(ctx); err != nil {
        log.Error("charge failed", logger.Err(err))
        return err
    }

    log.Info("order complete", logger.String("order_id", orderID))
    return nil
}

// Deep service layer — still has request_id
func chargeCard(ctx context.Context) error {
    log := logger.FromContext(ctx)
    log.Info("charging card")
    return nil
}
```

All three functions produce logs that share the same `request_id`, so you can reconstruct the full trace:

```json
{"request_id":"f4a1b2c3","msg":"order request received"}
{"request_id":"f4a1b2c3","msg":"processing order","order_id":"ORD-001"}
{"request_id":"f4a1b2c3","msg":"charging card"}
{"request_id":"f4a1b2c3","msg":"order complete","order_id":"ORD-001"}
{"request_id":"f4a1b2c3","msg":"request completed","status":201,"latency":"8ms"}
```

---

## Config Reference

| Environment Variable    | Type    | Default            | Description                              |
|-------------------------|---------|--------------------|------------------------------------------|
| `LOGGER_ENV`            | string  | `production`       | `development` or `production`            |
| `LOGGER_LEVEL`          | string  | `info`             | `debug`, `info`, `warn`, `error`, `fatal`|
| `LOGGER_CALLER`         | bool    | `false`            | Include `caller` field (file:line)       |
| `LOGGER_CONSOLE`        | bool    | `true`             | Write to stdout                          |
| `LOGGER_FILE`           | bool    | `true`             | Write to rotating file                   |
| `LOGGER_FILE_PATH`      | string  | `./logs/app.log`   | Path to log file                         |
| `LOGGER_MAX_SIZE_MB`    | int     | `100`              | Max file size before rotation (MB)       |
| `LOGGER_MAX_BACKUPS`    | int     | `30`               | Max number of old log files to keep      |
| `LOGGER_MAX_AGE_DAYS`   | int     | `30`               | Max age of old log files (days)          |
| `LOGGER_COMPRESS`       | bool    | `false`            | Gzip-compress rotated files              |

---

## Middleware Log Fields Reference

Fields logged on every request:

| Field           | Type     | Description                                        |
|-----------------|----------|----------------------------------------------------|
| `request_id`    | string   | UUID v4, read from `X-Request-ID` or generated     |
| `method`        | string   | HTTP method (`GET`, `POST`, …)                     |
| `path`          | string   | Request path (route pattern where available)       |
| `query`         | string   | Query string with sensitive values redacted        |
| `ip`            | string   | Client IP (checks X-Forwarded-For, X-Real-IP)      |
| `user_agent`    | string   | `User-Agent` header                                |
| `content_type`  | string   | `Content-Type` request header                      |
| `request_size`  | int      | `Content-Length` in bytes (-1 if unknown)          |
| `status`        | int      | HTTP response status code                          |
| `response_size` | int      | Actual bytes written in response body              |
| `latency`       | duration | Total handler execution time                       |
| `uploaded_files`| array    | File metadata for multipart uploads (name, size, content_type) — only present when content-type is multipart/form-data |
