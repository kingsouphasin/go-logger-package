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
//	import "github.com/kingsouphasin/logger"
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
//	LOGGER_MAX_AGE_DAYS max age of old files in days (default: 30)
//	LOGGER_COMPRESS     gzip rotated files (default: false)
//
// # Log rotation
//
// Logs rotate on whichever comes first: the file reaching MaxSizeMB, or
// midnight (daily). Old files are cleaned up based on MaxBackups and
// MaxAgeDays.
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
//	github.com/kingsouphasin/logger/middleware/http   — net/http
//	github.com/kingsouphasin/logger/middleware/gin    — Gin
//	github.com/kingsouphasin/logger/middleware/echo   — Echo
//	github.com/kingsouphasin/logger/middleware/fiber  — Fiber
//	github.com/kingsouphasin/logger/middleware/chi    — Chi
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
