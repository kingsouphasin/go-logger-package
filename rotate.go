package logger

import "gopkg.in/natefinch/lumberjack.v2"

func newRotatingWriter(cfg Config) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}
