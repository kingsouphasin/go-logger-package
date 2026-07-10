# logger

A production-ready Go structured logger built on [zap](https://github.com/uber-go/zap), with dual console + file output, automatic log rotation, and drop-in middleware for the most common Go HTTP frameworks.

```
github.com/kingsouphasin/go-logger-package
```

---

## Features

- **Zero-config** — works out of the box with sensible defaults
- **Dual output** — writes to console and a rotating file simultaneously
- **Log rotation** — size-based, with gzip compression of old files
- **Environment-driven config** — all settings via `.env` or environment variables
- **Development & production modes** — colored console output in dev, JSON in prod
- **Context propagation** — carry a logger through `context.Context`
- **Framework middleware** — net/http, Gin, Echo, Fiber, Chi (each an independent module)
- **Request enrichment** — auto-logs `request_id`, IP, user-agent, sizes, latency
- **`Handle` wrapper** — logger injected directly into handlers, no `FromContext` call needed
- **File upload safety** — logs multipart file metadata (name, size) but never file content
- **Opt-in body logging** — capture request/response bodies with JSON key redaction and truncation
- **Sensitive query redaction** — tokens, API keys, passwords are automatically masked

---

## Repository Structure

```
github.com/kingsouphasin/go-logger-package          ← core package (no framework deps)
├── middleware/
│   ├── http/                                        ← net/http middleware
│   ├── gin/                                         ← Gin middleware
│   ├── echo/                                        ← Echo middleware
│   ├── fiber/                                       ← Fiber middleware
│   └── chi/                                         ← Chi middleware
└── examples/
    ├── hello-world/                                 ← CLI demo: all logger features
    └── http-server/                                 ← HTTP server demo: middleware + request_id
```

Each middleware lives in its own Go module, so importing the core package does not pull in any framework dependencies.

---

## Installation

```bash
go get github.com/kingsouphasin/go-logger-package
```

Install only the middleware for the framework you use:

```bash
go get github.com/kingsouphasin/go-logger-package/middleware/http    # net/http
go get github.com/kingsouphasin/go-logger-package/middleware/gin     # Gin
go get github.com/kingsouphasin/go-logger-package/middleware/echo    # Echo
go get github.com/kingsouphasin/go-logger-package/middleware/fiber   # Fiber
go get github.com/kingsouphasin/go-logger-package/middleware/chi     # Chi
```

> **Import alias required:** because the module path ends in `go-logger-package` instead of `logger`, you must use an explicit alias in your import:
> ```go
> import logger "github.com/kingsouphasin/go-logger-package"
> ```

---

## Runnable Examples

```bash
# CLI demo — all logger features, no HTTP
cd examples/hello-world && go run main.go

# HTTP server — automatic request_id on every log line
cd examples/http-server && go run main.go
curl http://localhost:8080/hello?name=world
curl -H "X-Request-ID: my-trace-id" http://localhost:8080/order
```

---

## Understanding `request_id`

The logger has two distinct modes:

### Global logger — for app-level events (no `request_id`)

```go
import logger "github.com/kingsouphasin/go-logger-package"

logger.Info("server started", logger.String("port", "8080"))
```

- Available everywhere, zero setup
- **No `request_id`** — there is no HTTP request here, so there is nothing to identify
- Use for: startup, shutdown, background jobs, cron tasks

### Context logger — for request-scoped events (has `request_id`)

```go
log := logger.FromContext(ctx)  // inside a handler — see middleware section
log.Info("processing order")    // automatically includes request_id + all request fields
```

- Retrieved from `context.Context` inside an HTTP handler
- **Has `request_id`** — injected by the middleware before your handler runs
- Use for: everything that happens inside an HTTP request

> **Rule of thumb:** use `logger.Info(...)` for app events. Use `logger.FromContext(ctx)` (or `Handle`) inside handlers and any service functions they call.

### How `request_id` flows through a request

```
HTTP Request arrives
        │
        ▼
┌─────────────────────────────────────────────────────┐
│  Middleware (runs before your handler)              │
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
│  ginlogger.Handle(func(c *gin.Context, log Logger)  │
│      log.Info("ok")  ← has request_id              │
│      processOrder(c.Request.Context(), id)          │
│          log := logger.FromContext(ctx)             │
│          log.Info("done") ← still has request_id   │
└─────────────────────────────────────────────────────┘
        │
        ▼
Middleware logs "request completed" (status, latency, response_size)
```

---

## Quick Start

### Zero-config global logger

No setup required. Import and log:

```go
package main

import logger "github.com/kingsouphasin/go-logger-package"

func main() {
    defer logger.Sync()

    logger.Info("server started", logger.String("port", "8080"))
    logger.Warn("high memory usage", logger.Int("mb", 512))
    logger.Error("failed to connect", logger.String("host", "db.internal"))
}
```

The global logger initializes automatically on import, reading config from `.env` and environment variables.

### Sugared (key-value) style

```go
logger.Infow("user signed in", "user_id", 42, "email", "user@example.com")
logger.Errorw("payment failed", "order_id", "ORD-001", "reason", "card declined")
```

---

## Configuration

Create a `.env` file in your project root. All fields are optional — defaults shown:

```env
# Output format: "development" = colored console, "production" = JSON (default)
LOGGER_ENV=production

# Minimum level: debug | info | warn | error | fatal
LOGGER_LEVEL=info

# Add caller file:line to every log entry
LOGGER_CALLER=false

# Write to stdout
LOGGER_CONSOLE=true

# Write to a rotating file
LOGGER_FILE=true
LOGGER_FILE_PATH=./logs/app.log

# Rotate when file reaches this size (MB)
LOGGER_MAX_SIZE_MB=100

# How many rotated files to keep
LOGGER_MAX_BACKUPS=30

# Delete rotated files older than N days
LOGGER_MAX_AGE_DAYS=30

# Gzip-compress rotated files
LOGGER_COMPRESS=true

# Log HTTP request/response body content (middleware). Off by default.
LOGGER_LOG_BODY=false

# Max bytes of a body to log when LOGGER_LOG_BODY is enabled (truncated beyond this)
LOGGER_MAX_BODY_BYTES=4096
```

### Log rotation

Rotation triggers when the file reaches `LOGGER_MAX_SIZE_MB`. The current file is
renamed to a timestamped backup (gzip-compressed when `LOGGER_COMPRESS=true`) and a
fresh log file is created. Old files are cleaned up based on `LOGGER_MAX_BACKUPS` and
`LOGGER_MAX_AGE_DAYS`.

> **Run one instance per log file.** Like every Go file-rotating logger, this package
> is not multi-process safe — two processes writing the same file will corrupt each
> other's rotation. In dev, make sure a previous run has fully exited before starting a new one.

### HTTP body logging

Set `LOGGER_LOG_BODY=true` to have the middleware log request and response **body
content** as `request_body` / `response_body` fields. Safety rules:

- **Off by default** — no capture, no cost unless enabled.
- **Text/JSON only** — `application/json`, `text/*`, `x-www-form-urlencoded`. Multipart
  and binary are skipped (uploads are logged as file metadata, as before).
- **Redaction** — JSON bodies have sensitive keys (`password`, `token`, `secret`,
  `signature`, `sign`, …) replaced with `[redacted]`, recursively.
- **Truncation** — bodies longer than `LOGGER_MAX_BODY_BYTES` (default 4096) are cut with
  a `...[truncated]` marker.

---

## Custom Instance

Use `New()` when you need separate settings for a specific component or test:

```go
package main

import (
    "fmt"
    logger "github.com/kingsouphasin/go-logger-package"
)

func main() {
    l, err := logger.New(logger.Config{
        Env:        "development", // colored console output
        Level:      "debug",
        Console:    true,
        File:       true,
        FilePath:   "./logs/myapp.log",
        MaxSizeMB:  50,
        MaxBackups: 7,
        MaxAgeDays: 7,
        Compress:   true,
        Caller:     true,
        LogBody:      true, // capture request/response bodies (middleware)
        MaxBodyBytes: 4096,
    })
    if err != nil {
        fmt.Println("failed to init logger:", err)
        return
    }
    defer l.Sync()

    l.Info("custom logger ready")

    // Replace the global default so logger.Info(...) uses this config too
    logger.SetDefault(l)
}
```

---

## Structured Fields

The package re-exports common zap field helpers so you never need to import zap directly:

```go
import (
    "time"
    logger "github.com/kingsouphasin/go-logger-package"
)

logger.Info("order placed",
    logger.String("order_id", "ORD-123"),
    logger.Int("items", 3),
    logger.Float64("total", 99.95),
    logger.Bool("paid", true),
    logger.Duration("processing", 320*time.Millisecond),
    logger.Any("tags", []string{"promo", "express"}),
    logger.Err(err),
)
```

### Child loggers with `With()`

Attach fixed fields that appear on every subsequent log from that logger:

```go
userLog := logger.With(
    logger.String("user_id", "u-42"),
    logger.String("role", "admin"),
)
userLog.Info("profile viewed")      // includes user_id and role
userLog.Warn("suspicious login")    // includes user_id and role
```

### Named loggers

Add a component prefix to distinguish logs from different parts of your app:

```go
dbLog  := logger.Named("database")
dbLog.Info("connected")             // {"logger":"database","msg":"connected"}

payLog := logger.Named("payment")
payLog.Info("charge initiated")     // {"logger":"payment","msg":"charge initiated"}
```

### Dynamic log level

Change the minimum log level at runtime without restarting:

```go
logger.SetLevel("debug")   // enable verbose logging
logger.SetLevel("warn")    // suppress info and debug
```

---

## Context Integration

Store a logger in `context.Context` and retrieve it anywhere downstream — no need to pass a logger as a function parameter:

```go
import (
    "context"
    logger "github.com/kingsouphasin/go-logger-package"
)

// Store (the middleware does this automatically for HTTP handlers)
ctx := logger.WithContext(r.Context(), log)

// Retrieve anywhere downstream
func processPayment(ctx context.Context, amount float64) {
    log := logger.FromContext(ctx)
    log.Info("charging card", logger.Float64("amount", amount))
}
```

`FromContext` falls back to the global default logger if no logger is stored in the context.

---

## Middleware

Every middleware package provides two things:

- **`Middleware()`** — registers the logger middleware on the router
- **`Handle(fn)`** — wraps your handler and injects the logger as a parameter (no `FromContext` needed)

| Framework | Handle signature |
|-----------|-----------------|
| net/http | `httplogger.Handle(func(http.ResponseWriter, *http.Request, logger.Logger))` |
| Gin      | `ginlogger.Handle(func(*gin.Context, logger.Logger))` |
| Echo     | `echologger.Handle(func(echo.Context, logger.Logger) error)` |
| Fiber    | `fiberlogger.Handle(func(*fiber.Ctx, logger.Logger) error)` |
| Chi      | `chilogger.Handle(func(http.ResponseWriter, *http.Request, logger.Logger))` |

### net/http

```go
package main

import (
    "fmt"
    "net/http"

    logger "github.com/kingsouphasin/go-logger-package"
    httplogger "github.com/kingsouphasin/go-logger-package/middleware/http"
)

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("/users", httplogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        log.Info("listing users")
        fmt.Fprintln(w, `{"users":[]}`)
    }))

    http.ListenAndServe(":8080", httplogger.Middleware()(mux))
}
```

### Gin

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
    logger "github.com/kingsouphasin/go-logger-package"
    ginlogger "github.com/kingsouphasin/go-logger-package/middleware/gin"
)

func main() {
    r := gin.New()
    r.Use(ginlogger.Middleware())

    r.GET("/users/:id", ginlogger.Handle(func(c *gin.Context, log logger.Logger) {
        id := c.Param("id")
        log.Info("fetching user", logger.String("id", id))
        c.JSON(http.StatusOK, gin.H{"id": id})
    }))

    r.Run(":8080")
}
```

### Echo

```go
package main

import (
    "net/http"

    logger "github.com/kingsouphasin/go-logger-package"
    echologger "github.com/kingsouphasin/go-logger-package/middleware/echo"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()
    e.Use(echologger.Middleware())

    e.GET("/users/:id", echologger.Handle(func(c echo.Context, log logger.Logger) error {
        id := c.Param("id")
        log.Info("fetching user", logger.String("id", id))
        return c.JSON(http.StatusOK, map[string]string{"id": id})
    }))

    e.Start(":8080")
}
```

### Fiber

Fiber uses [fasthttp](https://github.com/valyala/fasthttp) which is not compatible with standard `context.Context`. Use `Handle` or `FromFiberCtx` — do **not** call `logger.FromContext` in Fiber handlers:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    logger "github.com/kingsouphasin/go-logger-package"
    fiberlogger "github.com/kingsouphasin/go-logger-package/middleware/fiber"
)

func main() {
    app := fiber.New()
    app.Use(fiberlogger.Middleware())

    app.Get("/users/:id", fiberlogger.Handle(func(c *fiber.Ctx, log logger.Logger) error {
        id := c.Params("id")
        log.Info("fetching user", logger.String("id", id))
        return c.JSON(fiber.Map{"id": id})
    }))

    app.Listen(":8080")
}
```

### Chi

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
    logger "github.com/kingsouphasin/go-logger-package"
    chilogger "github.com/kingsouphasin/go-logger-package/middleware/chi"
)

func main() {
    r := chi.NewRouter()
    r.Use(chilogger.Middleware)

    r.Get("/users/{id}", chilogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        id := chi.URLParam(r, "id")
        log.Info("fetching user", logger.String("id", id))
        fmt.Fprintf(w, `{"id":"%s"}`, id)
    }))

    http.ListenAndServe(":8080", r)
}
```

### Passing the logger into service layers

When your handler calls service functions, pass `ctx` down — each function retrieves the same logger (with the same `request_id`) via `FromContext`:

```go
r.GET("/orders", ginlogger.Handle(func(c *gin.Context, log logger.Logger) {
    log.Info("order request received")

    if err := placeOrder(c.Request.Context(), "ORD-001"); err != nil {
        log.Error("order failed", logger.Err(err))
        c.JSON(500, gin.H{"error": "internal"})
        return
    }
    c.JSON(201, gin.H{"status": "created"})
}))

func placeOrder(ctx context.Context, orderID string) error {
    log := logger.FromContext(ctx)          // same request_id as the handler
    log.Info("placing order", logger.String("order_id", orderID))
    return chargeCard(ctx)
}

func chargeCard(ctx context.Context) error {
    log := logger.FromContext(ctx)          // same request_id, three levels deep
    log.Info("charging card")
    return nil
}
```

---

## Tracing a Request End-to-End

All log lines from a single request share the same `request_id`, making it easy to filter in any log aggregator (Datadog, Loki, CloudWatch):

```json
{"request_id":"f4a1b2c3","msg":"order request received"}
{"request_id":"f4a1b2c3","msg":"placing order","order_id":"ORD-001"}
{"request_id":"f4a1b2c3","msg":"charging card"}
{"request_id":"f4a1b2c3","msg":"request completed","status":201,"latency":"8ms"}
```

---

## Log Output Examples

### Development mode (`LOGGER_ENV=development`)

Colored, human-readable — ideal for local development:

```
2026-06-15T10:30:00.000+0700  INFO  server listening  {"addr": ":8080"}
2026-06-15T10:30:01.000+0700  INFO  order request received  {"request_id": "f4a1b2c3", "method": "POST", "path": "/orders", "ip": "127.0.0.1"}
2026-06-15T10:30:01.008+0700  INFO  request completed  {"request_id": "f4a1b2c3", "status": 201, "latency": "8ms"}
```

### Production mode (`LOGGER_ENV=production`)

JSON — one object per line, ready for Datadog, Loki, CloudWatch:

```json
{"level":"info","ts":"2026-06-15T10:30:01.000Z","msg":"order request received","request_id":"f4a1b2c3","method":"POST","path":"/orders","query":"","ip":"203.0.113.5","user_agent":"curl/8.1.2","content_type":"application/json","request_size":42}
{"level":"info","ts":"2026-06-15T10:30:01.008Z","msg":"request completed","request_id":"f4a1b2c3","status":201,"response_size":24,"latency":"8ms"}
```

### File upload (`multipart/form-data`)

File metadata is logged — file content is never logged:

```json
{
  "msg": "request completed",
  "request_id": "f9e8d7c6",
  "method": "POST",
  "path": "/upload",
  "uploaded_files": [
    {"name": "photo.jpg", "size": 204800, "content_type": "image/jpeg"},
    {"name": "doc.pdf",   "size": 512000, "content_type": "application/pdf"}
  ],
  "status": 200,
  "latency": "45ms"
}
```

### Sensitive query parameter redaction

`token`, `api_key`, `password`, `secret`, `code`, `authorization`, `access_token`, `key`, `state`, and `client_secret` are automatically masked:

```
Request URL:  /search?q=golang&api_key=my-secret&page=2
Logged query: api_key=%5Bredacted%5D&page=2&q=golang
```

### Body logging (`LOGGER_LOG_BODY=true`)

Request/response bodies are logged with JSON sensitive keys redacted and content
truncated to `LOGGER_MAX_BODY_BYTES`:

```
Request body: {"user":"alice","password":"secret","amount":100}
Logged:       "request_body": "{\"amount\":100,\"password\":\"[redacted]\",\"user\":\"alice\"}"
```

---

## Config Reference

| Environment Variable    | Type    | Default            | Description                               |
|-------------------------|---------|--------------------|-------------------------------------------|
| `LOGGER_ENV`            | string  | `production`       | `development` or `production`             |
| `LOGGER_LEVEL`          | string  | `info`             | `debug`, `info`, `warn`, `error`, `fatal` |
| `LOGGER_CALLER`         | bool    | `false`            | Include `caller` field (file:line)        |
| `LOGGER_CONSOLE`        | bool    | `true`             | Write to stdout                           |
| `LOGGER_FILE`           | bool    | `true`             | Write to rotating file                    |
| `LOGGER_FILE_PATH`      | string  | `./logs/app.log`   | Path to log file                          |
| `LOGGER_MAX_SIZE_MB`    | int     | `100`              | Max file size before rotation (MB)        |
| `LOGGER_MAX_BACKUPS`    | int     | `30`               | Max number of old log files to keep       |
| `LOGGER_MAX_AGE_DAYS`   | int     | `30`               | Max age of old log files (days)           |
| `LOGGER_COMPRESS`       | bool    | `true`             | Gzip-compress rotated files               |
| `LOGGER_LOG_BODY`       | bool    | `false`            | Log HTTP request/response body (middleware) |
| `LOGGER_MAX_BODY_BYTES` | int     | `4096`             | Max body bytes to log when body logging on |

---

## Middleware Log Fields Reference

Every middleware logs these fields automatically on each request:

| Field           | Type     | Source                                              |
|-----------------|----------|-----------------------------------------------------|
| `request_id`    | string   | `X-Request-ID` header, or generated UUID v4         |
| `method`        | string   | HTTP method (`GET`, `POST`, …)                      |
| `path`          | string   | Route pattern (e.g. `/users/:id`) where supported   |
| `query`         | string   | Query string with sensitive values redacted         |
| `ip`            | string   | Checks `X-Forwarded-For`, `X-Real-IP`, `RemoteAddr` |
| `user_agent`    | string   | `User-Agent` request header                         |
| `content_type`  | string   | `Content-Type` request header                       |
| `request_size`  | int      | `Content-Length` in bytes (`-1` if unknown)         |
| `status`        | int      | HTTP response status code                           |
| `response_size` | int      | Actual bytes written in the response body           |
| `latency`       | duration | Total time from request received to response sent   |
| `uploaded_files`| array    | Only for `multipart/form-data` — `name`, `size`, `content_type` per file |
| `request_body`  | string   | Only when `LOGGER_LOG_BODY=true` — request body, redacted + truncated (text/JSON only) |
| `response_body` | string   | Only when `LOGGER_LOG_BODY=true` — response body, redacted + truncated (text/JSON only) |
