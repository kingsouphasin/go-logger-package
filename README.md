# logger

A production-ready Go structured logger built on [zap](https://github.com/uber-go/zap), with dual console + file output, automatic log rotation, and drop-in middleware for the most common Go HTTP frameworks.

```
github.com/kingsouphasin/go-logger-package
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
- **`Handle` wrapper** — logger injected directly into handlers, no `FromContext` call needed
- **File upload safety** — logs multipart file metadata (name, size) but never file content
- **Sensitive query redaction** — tokens, API keys, passwords are automatically masked

---

## Repository Structure

```
github.com/kingsouphasin/go-logger-package        ← core package (zero dependencies on any framework)
├── middleware/
│   ├── http/                          ← net/http middleware
│   ├── gin/                           ← Gin middleware
│   ├── echo/                          ← Echo middleware
│   ├── fiber/                         ← Fiber middleware
│   └── chi/                           ← Chi middleware
└── examples/
    ├── hello-world/                   ← CLI demo: all logger features, no HTTP
    └── http-server/                   ← HTTP server demo: middleware + request_id
```

Each middleware lives in its own Go module, so importing only the core package does not pull in any framework dependencies.

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

---

## Runnable Examples

Clone the repo and run either example to see the package in action:

```bash
# All logger features in a CLI program (no HTTP, no request_id)
cd examples/hello-world
go run main.go

# HTTP server with middleware — shows automatic request_id on every log line
cd examples/http-server
go run main.go
# then in another terminal:
curl http://localhost:8080/hello?name=world
curl -H "X-Request-ID: my-trace-id" http://localhost:8080/order
```

---

## Understanding `request_id`

This is the most important concept in the package. The logger has two distinct modes:

### Global logger — for app-level events (no `request_id`)

```go
logger.Info("server started", logger.String("port", "8080"))
```

- Available everywhere with zero setup
- **Does not have `request_id`** — there is no HTTP request at this point, so there is nothing to identify
- Use this for: startup, shutdown, background jobs, cron tasks

### Context logger — for request-scoped events (has `request_id`)

```go
log := logger.FromContext(ctx)   // or use Handle — see below
log.Info("processing order")     // automatically includes request_id and all request fields
```

- Retrieved from `context.Context` inside an HTTP handler
- **Has `request_id`** — the middleware generates/reads it and attaches it before your handler runs
- Use this for: everything that happens inside an HTTP request

> **Rule of thumb:** use `logger.Info(...)` for app events. Use `logger.FromContext(ctx)` (or `Handle`) inside HTTP handlers and any service functions they call.

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
│  // Option A — logger injected by Handle wrapper   │
│  app.Get("/", handler.Handle(func(c, log) { ... }) │
│                                                     │
│  // Option B — retrieve from context manually      │
│  log := logger.FromContext(r.Context())             │
│                                                     │
│  log.Info("doing work")   ← has request_id         │
│  processOrder(ctx, id)    ← pass ctx down           │
│    log := FromContext(ctx)                          │
│    log.Info("order done") ← still has request_id   │
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

import "github.com/kingsouphasin/go-logger-package"

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

Create a `.env` file in your project root. All fields are optional — defaults shown below:

```env
# Output format: "development" = colored console, "production" = JSON (default)
LOGGER_ENV=production

# Minimum level to emit: debug | info | warn | error | fatal
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
LOGGER_COMPRESS=false
```

### Log rotation

Rotation is triggered by whichever comes first: the file reaching `LOGGER_MAX_SIZE_MB`, or midnight (daily). Old files are cleaned up automatically based on `LOGGER_MAX_BACKUPS` and `LOGGER_MAX_AGE_DAYS`.

---

## Custom Instance

Use `New()` when you need separate settings for a specific component or test:

```go
package main

import (
    "fmt"
    "github.com/kingsouphasin/go-logger-package"
)

func main() {
    l, err := logger.New(logger.Config{
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
    })
    if err != nil {
        fmt.Println("failed to init logger:", err)
        return
    }
    defer l.Sync()

    l.Info("custom logger ready")

    // Optionally replace the global default so logger.Info(...) also uses this config
    logger.SetDefault(l)
}
```

---

## Structured Fields

The package re-exports common zap field helpers so you never need to import zap directly:

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

### Child loggers with `With()`

Attach fixed fields that appear on every subsequent log from that logger:

```go
userLog := logger.With(logger.String("user_id", "u-42"), logger.String("role", "admin"))
userLog.Info("profile viewed")          // includes user_id and role
userLog.Warn("suspicious login")        // includes user_id and role
```

### Named loggers

Add a component name prefix to distinguish logs from different parts of your app:

```go
dbLog := logger.Named("database")
dbLog.Info("connected")                 // {"logger":"database","msg":"connected"}

payLog := logger.Named("payment")
payLog.Info("charge initiated")         // {"logger":"payment","msg":"charge initiated"}
```

### Dynamic log level

Change the minimum log level at runtime without restarting:

```go
logger.SetLevel("debug")    // enable verbose logging
logger.SetLevel("warn")     // suppress info and debug
```

---

## Context Integration

Store a logger in `context.Context` and retrieve it anywhere downstream — no need to pass a logger as a function parameter:

```go
// Store a logger in context (the middleware does this for you in HTTP handlers)
ctx := logger.WithContext(r.Context(), log)

// Retrieve it anywhere — in service functions, repositories, etc.
log := logger.FromContext(ctx)
log.Info("processing payment")      // inherits all fields including request_id
```

`FromContext` falls back to the global default logger if no logger is stored in the context.

---

## Middleware

### Using `Handle` — the recommended approach

Every middleware package exports a `Handle` wrapper that injects the logger as a function parameter. You write a handler that accepts a `logger.Logger` directly — no `FromContext` call needed:

| Framework | Handle signature |
|-----------|-----------------|
| net/http | `httplogger.Handle(func(http.ResponseWriter, *http.Request, logger.Logger))` |
| Gin | `ginlogger.Handle(func(*gin.Context, logger.Logger))` |
| Echo | `echologger.Handle(func(echo.Context, logger.Logger) error)` |
| Fiber | `fiberlogger.Handle(func(*fiber.Ctx, logger.Logger) error)` |
| Chi | `chilogger.Handle(func(http.ResponseWriter, *http.Request, logger.Logger))` |

`Handle` requires the middleware to be registered first (`r.Use(ginlogger.Middleware())`). It only handles the logger extraction step.

### net/http

```go
package main

import (
    "net/http"

    "github.com/kingsouphasin/go-logger-package"
    httplogger "github.com/kingsouphasin/go-logger-package/middleware/http"
)

func main() {
    mux := http.NewServeMux()

    // Recommended: Handle injects the logger automatically
    mux.HandleFunc("/hello", httplogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        log.Info("handling request")
        w.Write([]byte("OK"))
    }))

    // Alternative: retrieve from context manually
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
    "github.com/kingsouphasin/go-logger-package"
    ginlogger "github.com/kingsouphasin/go-logger-package/middleware/gin"
)

func main() {
    r := gin.New()
    r.Use(ginlogger.Middleware())

    // Recommended: Handle injects the logger automatically
    r.GET("/users/:id", ginlogger.Handle(func(c *gin.Context, log logger.Logger) {
        log.Info("fetching user", logger.String("id", c.Param("id")))
        c.JSON(200, gin.H{"id": c.Param("id")})
    }))

    // Alternative: retrieve from context manually
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

    "github.com/kingsouphasin/go-logger-package"
    echologger "github.com/kingsouphasin/go-logger-package/middleware/echo"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()
    e.Use(echologger.Middleware())

    // Recommended: Handle injects the logger automatically
    e.GET("/users/:id", echologger.Handle(func(c echo.Context, log logger.Logger) error {
        log.Info("fetching user", logger.String("id", c.Param("id")))
        return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
    }))

    // Alternative: retrieve from context manually
    e.GET("/users2/:id", func(c echo.Context) error {
        log := logger.FromContext(c.Request().Context())
        log.Info("fetching user", logger.String("id", c.Param("id")))
        return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
    })

    e.Start(":8080")
}
```

### Fiber

Fiber uses [fasthttp](https://github.com/valyala/fasthttp) which is not compatible with standard `context.Context`. Use `Handle` or `FromFiberCtx` — do **not** use `logger.FromContext` in Fiber handlers:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/kingsouphasin/go-logger-package"
    fiberlogger "github.com/kingsouphasin/go-logger-package/middleware/fiber"
)

func main() {
    app := fiber.New()
    app.Use(fiberlogger.Middleware())

    // Recommended: Handle injects the logger automatically
    app.Get("/users/:id", fiberlogger.Handle(func(c *fiber.Ctx, log logger.Logger) error {
        log.Info("fetching user", logger.String("id", c.Params("id")))
        return c.JSON(fiber.Map{"id": c.Params("id")})
    }))

    // Alternative: use FromFiberCtx manually (Fiber-specific, not FromContext)
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
    "github.com/kingsouphasin/go-logger-package"
    chilogger "github.com/kingsouphasin/go-logger-package/middleware/chi"
)

func main() {
    r := chi.NewRouter()
    r.Use(chilogger.Middleware)

    // Recommended: Handle injects the logger automatically
    r.Get("/users/{id}", chilogger.Handle(func(w http.ResponseWriter, r *http.Request, log logger.Logger) {
        log.Info("fetching user", logger.String("id", chi.URLParam(r, "id")))
        w.Write([]byte("OK"))
    }))

    // Alternative: retrieve from context manually
    r.Get("/users2/{id}", func(w http.ResponseWriter, r *http.Request) {
        log := logger.FromContext(r.Context())
        log.Info("fetching user", logger.String("id", chi.URLParam(r, "id")))
        w.Write([]byte("OK"))
    })

    http.ListenAndServe(":8080", r)
}
```

---

## Tracing a Request End-to-End

Every log line emitted via `logger.FromContext(ctx)` (or `Handle`) automatically carries `request_id`. Pass `ctx` down through all service layers and every log line will be linked to the same request.

```go
// Handler
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

// Service layer — receives ctx, inherits request_id automatically
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

// Deep layer — still has request_id
func chargeCard(ctx context.Context) error {
    log := logger.FromContext(ctx)
    log.Info("charging card")
    return nil
}
```

All log lines share the same `request_id` and can be filtered together in any log aggregator:

```json
{"request_id":"f4a1b2c3","msg":"order request received"}
{"request_id":"f4a1b2c3","msg":"processing order","order_id":"ORD-001"}
{"request_id":"f4a1b2c3","msg":"charging card"}
{"request_id":"f4a1b2c3","msg":"order complete","order_id":"ORD-001"}
{"request_id":"f4a1b2c3","msg":"request completed","status":201,"latency":"8ms"}
```

---

## Log Output Examples

### Development mode (`LOGGER_ENV=development`)

Colored, human-readable output — ideal for local development:

```
2024-01-15T10:30:00.000Z  INFO  server listening  {"addr": ":8080"}
2024-01-15T10:30:01.000Z  INFO  order request received  {"request_id": "f4a1b2c3", "method": "POST", "path": "/order", "ip": "127.0.0.1"}
2024-01-15T10:30:01.008Z  INFO  request completed  {"request_id": "f4a1b2c3", "status": 201, "latency": "8ms"}
```

### Production mode (`LOGGER_ENV=production`)

JSON — one object per line, ready for Datadog, Loki, CloudWatch, etc.:

```json
{"level":"info","ts":"2024-01-15T10:30:01.000Z","msg":"order request received","request_id":"f4a1b2c3","method":"POST","path":"/order","query":"","ip":"203.0.113.5","user_agent":"curl/8.1.2","content_type":"application/json","request_size":42}
{"level":"info","ts":"2024-01-15T10:30:01.008Z","msg":"request completed","request_id":"f4a1b2c3","status":201,"response_size":38,"latency":"8ms"}
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

Parameters such as `token`, `api_key`, `password`, `secret`, `code`, `authorization`, `access_token`, `key`, `state`, and `client_secret` are automatically masked:

```
Request URL:  /search?q=golang&api_key=my-secret&page=2
Logged query: api_key=%5Bredacted%5D&page=2&q=golang
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

Every middleware logs these fields automatically on each request:

| Field           | Type     | Source                                             |
|-----------------|----------|----------------------------------------------------|
| `request_id`    | string   | `X-Request-ID` header, or generated UUID v4        |
| `method`        | string   | HTTP method (`GET`, `POST`, …)                     |
| `path`          | string   | Route pattern (e.g. `/users/:id`) where supported  |
| `query`         | string   | Query string with sensitive values redacted        |
| `ip`            | string   | Checks `X-Forwarded-For`, `X-Real-IP`, `RemoteAddr`|
| `user_agent`    | string   | `User-Agent` request header                        |
| `content_type`  | string   | `Content-Type` request header                      |
| `request_size`  | int      | `Content-Length` in bytes (`-1` if unknown)        |
| `status`        | int      | HTTP response status code                          |
| `response_size` | int      | Actual bytes written in the response body          |
| `latency`       | duration | Total time from request received to response sent  |
| `uploaded_files`| array    | Only for `multipart/form-data` — `name`, `size`, `content_type` per file |
