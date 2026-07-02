package logger

import (
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// writers holds one *lumberjack.Logger per absolute log file path. Sharing a
// single writer per file is essential: if two writers target the same file,
// each rotates it independently and they clobber each other's backups —
// producing several near-simultaneous rotations and empty log files.
var (
	writersMu sync.Mutex
	writers   = map[string]*lumberjack.Logger{}
)

func newRotatingWriter(cfg Config) *lumberjack.Logger {
	key, err := filepath.Abs(cfg.FilePath)
	if err != nil {
		key = cfg.FilePath
	}

	writersMu.Lock()
	defer writersMu.Unlock()

	if w, ok := writers[key]; ok {
		return w
	}

	w := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
	writers[key] = w
	return w
}
