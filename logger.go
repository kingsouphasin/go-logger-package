package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Fatalw(msg string, keysAndValues ...any)

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
		l, err = New(fallback)
	}
	if err != nil || l == nil {
		// Last resort: bare stderr logger so the package never has a nil defaultLogger
		z, _ := zap.NewProduction()
		l = &zapLogger{z: z, sugar: z.Sugar(), level: zap.NewAtomicLevelAt(zapcore.InfoLevel)}
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
