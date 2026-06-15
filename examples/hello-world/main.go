package main

import (
	"context"
	"errors"
	"time"

	"github.com/kingsouphasin/go-logger-package"
)

func main() {
	defer logger.Sync()

	// ─── 1. Zero-config global logger ────────────────────────────────────────
	logger.Info("=== hello-world logger example ===")

	logger.Debug("debug message (hidden at default info level)")
	logger.Info("application started")
	logger.Warn("low disk space", logger.Int("free_mb", 512))
	logger.Error("failed to connect", logger.String("host", "db.internal"), logger.Err(errors.New("connection refused")))

	// ─── 2. Sugared (key-value) style ────────────────────────────────────────
	logger.Info("--- sugared style ---")
	logger.Infow("user signed in",
		"user_id", 42,
		"email", "user@example.com",
		"ip", "203.0.113.5",
	)
	logger.Warnw("rate limit approaching",
		"user_id", 42,
		"requests", 95,
		"limit", 100,
	)

	// ─── 3. Structured fields ─────────────────────────────────────────────────
	logger.Info("--- structured fields ---")
	logger.Info("order placed",
		logger.String("order_id", "ORD-001"),
		logger.Int("items", 3),
		logger.Float64("total", 149.99),
		logger.Bool("paid", true),
		logger.Duration("checkout_duration", 320*time.Millisecond),
		logger.Any("tags", []string{"promo", "express"}),
	)

	// ─── 4. Child logger with With() ─────────────────────────────────────────
	logger.Info("--- child logger with With() ---")
	userLog := logger.With(
		logger.String("user_id", "u-42"),
		logger.String("role", "admin"),
	)
	userLog.Info("profile viewed")
	userLog.Warn("suspicious login attempt", logger.String("country", "XX"))
	userLog.Info("password changed")

	// ─── 5. Named logger ──────────────────────────────────────────────────────
	logger.Info("--- named logger ---")
	dbLog := logger.Named("database")
	dbLog.Info("connected", logger.String("dsn", "postgres://localhost:5432/app"))
	dbLog.Warn("slow query", logger.Duration("took", 1500*time.Millisecond), logger.String("query", "SELECT *"))

	payLog := logger.Named("payment")
	payLog.Info("charge initiated", logger.String("currency", "USD"), logger.Float64("amount", 99.00))

	// ─── 6. Context propagation ───────────────────────────────────────────────
	logger.Info("--- context propagation ---")

	// Simulate middleware storing a request-scoped logger in context
	requestLog := logger.With(
		logger.String("request_id", "req-abc123"),
		logger.String("method", "POST"),
		logger.String("path", "/checkout"),
	)
	ctx := logger.WithContext(context.Background(), requestLog)

	// Deep in your service code — no need to pass the logger as a parameter
	processOrder(ctx, "ORD-001")

	// ─── 7. Dynamic log level ────────────────────────────────────────────────
	logger.Info("--- dynamic level change ---")
	logger.Debug("this won't appear (level is info)")
	logger.SetLevel("debug")
	logger.Debug("now debug is enabled")
	logger.SetLevel("info")

	// ─── 8. Custom logger instance ────────────────────────────────────────────
	logger.Info("--- custom instance ---")
	custom, err := logger.New(logger.Config{
		Env:        "development",
		Level:      "debug",
		Console:    true,
		File:       false, // console only for this instance
		Caller:     true,
	})
	if err != nil {
		logger.Error("failed to create custom logger", logger.Err(err))
		return
	}
	defer custom.Sync()

	custom.Debug("verbose debug from custom logger")
	custom.Info("custom logger is ready", logger.String("env", "development"))

	logger.Info("=== example complete ===")
}

// processOrder simulates a service function that uses the context logger.
// It inherits request_id and all other fields from the middleware-level logger.
func processOrder(ctx context.Context, orderID string) {
	log := logger.FromContext(ctx)
	log.Info("processing order", logger.String("order_id", orderID))

	if err := chargeCard(ctx, 99.99); err != nil {
		log.Error("payment failed", logger.Err(err))
		return
	}

	log.Info("order dispatched", logger.String("order_id", orderID))
}

func chargeCard(ctx context.Context, amount float64) error {
	log := logger.FromContext(ctx)
	log.Info("charging card", logger.Float64("amount", amount))
	// simulate success
	return nil
}
