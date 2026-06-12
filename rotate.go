package logger

import (
	"context"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func newRotatingWriter(cfg Config) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

func startTimeRotation(ctx context.Context, lj *lumberjack.Logger) {
	go func() {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		timer := time.NewTimer(time.Until(next))
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				_ = lj.Rotate()
				next = next.Add(24 * time.Hour)
				timer.Reset(time.Until(next))
			}
		}
	}()
}
