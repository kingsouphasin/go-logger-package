// Package logger provides a production-ready structured logger built on
// [go.uber.org/zap], with dual console and file output, automatic log
// rotation, context propagation, and drop-in middleware for the most
// common Go HTTP frameworks.
//
// # Zero-config usage
//
// The package initializes a global default logger automatically on import.
// No setup is required:
//
//	import "github.com/kingsouphasin/go-logger-package"
//
//	logger.Info("server started", logger.String("port", "8080"))
//	logger.Warn("high memory", logger.Int("mb", 512))
//	logger.Error("db error", logger.Err(err))
//
// # Custom instance
//
// Use [New] when you need specific settings for a component or test:
//
//	l, err := logger.New(logger.Config{
//	    Env:      "development", // colored console output
//	    Level:    "debug",
//	    Console:  true,
//	    File:     true,
//	    FilePath: "./logs/app.log",
//	})
//
// # Configuration
//
// All settings are optional and can be provided via a .env file or
// environment variables. Supported variables:
//
//	LOGGER_ENV          development | production (default: production)
//	LOGGER_LEVEL        debug | info | warn | error | fatal (default: info)
//	LOGGER_CALLER       include caller file:line (default: false)
//	LOGGER_CONSOLE      write to stdout (default: true)
//	LOGGER_FILE         write to rotating file (default: true)
//	LOGGER_FILE_PATH    path to log file (default: ./logs/app.log)
//	LOGGER_MAX_SIZE_MB  rotate after N MB (default: 100)
//	LOGGER_MAX_BACKUPS  max old files to keep (default: 30)
//	LOGGER_MAX_AGE_DAYS   max age of old files in days (default: 30)
//	LOGGER_COMPRESS       gzip rotated files (default: true)
//	LOGGER_LOG_BODY       log HTTP request/response bodies, middleware (default: false)
//	LOGGER_MAX_BODY_BYTES max body bytes to log when enabled (default: 4096)
//
// # Log rotation
//
// Logs rotate when the file reaches MaxSizeMB: the current file is renamed to a
// timestamped backup (gzip-compressed when Compress is true) and a fresh file is
// created. Old files are cleaned up based on MaxBackups and MaxAgeDays. Run one
// process per log file — like every Go file-rotating logger, this package is not
// multi-process safe.
//
// # HTTP body logging
//
// Set LOGGER_LOG_BODY=true to have the middleware log request and response body
// content as request_body/response_body fields. Only text/JSON content types are
// captured (multipart and binary are skipped), JSON sensitive keys are redacted,
// and content is truncated to LOGGER_MAX_BODY_BYTES.
//
// # Output modes
//
// Setting Env to "development" produces colored, human-readable console
// output suited for local development. Setting it to "production" (the
// default) produces JSON output, one object per line, suitable for log
// aggregators such as Datadog, Loki, or CloudWatch.
//
// # Context propagation
//
// Store a logger in a [context.Context] to carry request-scoped fields
// (such as request_id) through your service layers without passing the
// logger as a function parameter:
//
//	// store
//	ctx := logger.WithContext(r.Context(), log)
//
//	// retrieve anywhere downstream
//	log := logger.FromContext(ctx)
//	log.Info("processing payment") // inherits all fields including request_id
//
// [FromContext] falls back to the global default logger if none is stored.
//
// # HTTP middleware
//
// Framework-specific middleware packages are available as separate modules
// so that importing the core package does not pull in framework dependencies:
//
//	github.com/kingsouphasin/go-logger-package/middleware/http   — net/http
//	github.com/kingsouphasin/go-logger-package/middleware/gin    — Gin
//	github.com/kingsouphasin/go-logger-package/middleware/echo   — Echo
//	github.com/kingsouphasin/go-logger-package/middleware/fiber  — Fiber
//	github.com/kingsouphasin/go-logger-package/middleware/chi    — Chi
//
// Each middleware automatically generates or propagates a request_id
// (read from the X-Request-ID header or generated as a UUID v4), attaches
// it to a child logger, stores that logger in context, and logs a
// "request completed" entry with status, latency, and response size.
//
// Use the Handle wrapper exported by each middleware package to receive the
// logger directly as a function parameter — no [FromContext] call needed:
//
//	r.Use(ginlogger.Middleware())
//
//	r.GET("/users/:id", ginlogger.Handle(func(c *gin.Context, log logger.Logger) {
//	    log.Info("fetching user", logger.String("id", c.Param("id")))
//	    c.JSON(200, gin.H{"id": c.Param("id")})
//	}))
//
// # Security
//
// Query parameters with sensitive names — including token, access_token,
// api_key, key, password, secret, client_secret, code, state, and
// authorization — are automatically redacted before logging.
//
// For multipart/form-data requests, file metadata (name, size,
// content-type) is logged but file content is never read or stored.
package logger
