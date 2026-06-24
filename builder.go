package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	z      *zap.Logger       // skip 1 — for Logger interface method calls
	zPkg   *zap.Logger       // skip 2 — for package-level global function calls
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

	// enc is safe to share across cores: zapcore.NewCore calls enc.Clone() internally.
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

	base := zap.New(zapcore.NewTee(cores...), zap.WithCaller(cfg.Caller))
	z    := base.WithOptions(zap.AddCallerSkip(1))
	zPkg := base.WithOptions(zap.AddCallerSkip(2))

	return &zapLogger{
		z:      z,
		zPkg:   zPkg,
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
func (l *zapLogger) Fatalw(msg string, kv ...any) { l.sugar.Fatalw(msg, kv...) }

func (l *zapLogger) With(fields ...zap.Field) Logger {
	z    := l.z.With(fields...)
	zPkg := l.zPkg.With(fields...)
	return &zapLogger{z: z, zPkg: zPkg, sugar: z.Sugar(), level: l.level, cancel: l.cancel}
}

func (l *zapLogger) Named(name string) Logger {
	z    := l.z.Named(name)
	zPkg := l.zPkg.Named(name)
	return &zapLogger{z: z, zPkg: zPkg, sugar: z.Sugar(), level: l.level, cancel: l.cancel}
}

// pkgZap returns the zap logger configured for package-level global function calls (skip 2).
func (l *zapLogger) pkgZap() *zap.Logger { return l.zPkg }

func (l *zapLogger) WithoutCaller() Logger {
	z    := l.z.WithOptions(zap.WithCaller(false))
	zPkg := l.zPkg.WithOptions(zap.WithCaller(false))
	return &zapLogger{z: z, zPkg: zPkg, sugar: z.Sugar(), level: l.level, cancel: l.cancel}
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
