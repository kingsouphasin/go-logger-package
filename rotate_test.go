package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Reproduces the reported bug: multiple independent lumberjack writers on the
// same file, under concurrent writes (as HTTP middleware produces), each rotate
// the over-limit file at nearly the same instant. The result is several backups
// created within milliseconds where all but one are EMPTY — exactly what was
// observed in production (app-...310/.316/.317.log all 0 bytes).
func TestBug_IndependentWritersClobber(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	// Pre-fill app.log with ~1.6MB of real data (over the 1MB limit).
	prefill(t, path, 1800)
	origSize := fileSize(t, path)
	t.Logf("pre-filled app.log = %d bytes", origSize)

	// Several independent writers (simulates package init() + user New() + more).
	mk := func() *lumberjack.Logger {
		return &lumberjack.Logger{Filename: path, MaxSize: 1, Compress: false}
	}
	writersList := []*lumberjack.Logger{mk(), mk(), mk(), mk()}

	// Fire concurrent writes across all writers simultaneously.
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, w := range writersList {
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func(w *lumberjack.Logger) {
				defer wg.Done()
				<-start
				w.Write([]byte("concurrent write\n"))
			}(w)
		}
	}
	close(start)
	wg.Wait()
	for _, w := range writersList {
		w.Close()
	}

	backups := backupFiles(t, dir)
	empty := 0
	dataPreserved := false
	t.Logf("backups created: %d", len(backups))
	for _, b := range backups {
		sz := fileSize(t, filepath.Join(dir, b))
		t.Logf("  backup %s = %d bytes", b, sz)
		if sz >= origSize {
			dataPreserved = true
		}
		if sz == 0 {
			empty++
		}
	}

	if len(backups) > 1 || empty > 0 || !dataPreserved {
		t.Logf("BUG REPRODUCED: %d backups, %d empty, dataPreserved=%v", len(backups), empty, dataPreserved)
	} else {
		t.Logf("did not reproduce this run (race timing); single writer would be safe")
	}
}

// Verifies the fix: newRotatingWriter returns a single shared writer per path,
// so only one rotation happens and the original data is preserved intact.
func TestFix_SharedWriterPreservesData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	prefill(t, path, 1800)
	origSize := fileSize(t, path)
	t.Logf("pre-filled app.log = %d bytes", origSize)

	cfg := Config{FilePath: path, MaxSizeMB: 1, Compress: false}

	// Simulate multiple New() calls to the same path (package init + user + more).
	handles := []*lumberjack.Logger{
		newRotatingWriter(cfg),
		newRotatingWriter(cfg),
		newRotatingWriter(cfg),
		newRotatingWriter(cfg),
	}
	for _, w := range handles {
		if w != handles[0] {
			t.Fatalf("expected a single shared writer for identical path, got distinct instances")
		}
	}

	// Same concurrent load that clobbered the independent writers above.
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, w := range handles {
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func(w *lumberjack.Logger) {
				defer wg.Done()
				<-start
				w.Write([]byte("concurrent write\n"))
			}(w)
		}
	}
	close(start)
	wg.Wait()
	handles[0].Close()

	backups := backupFiles(t, dir)
	t.Logf("backups created: %d", len(backups))
	for _, b := range backups {
		t.Logf("  backup %s = %d bytes", b, fileSize(t, filepath.Join(dir, b)))
	}

	if len(backups) != 1 {
		t.Fatalf("expected exactly 1 backup, got %d", len(backups))
	}
	if sz := fileSize(t, filepath.Join(dir, backups[0])); sz < origSize {
		t.Fatalf("data lost: backup is %d bytes, expected >= %d (original)", sz, origSize)
	}
	t.Logf("FIX VERIFIED: 1 backup, original %d bytes preserved", origSize)

	// clean registry entry so the test is isolated
	writersMu.Lock()
	delete(writers, mustAbs(path))
	writersMu.Unlock()
}

// prefill writes n lines of ~900 bytes each to path, producing a file well
// over the 1MB rotation limit used by the tests.
func prefill(t *testing.T, path string, n int) {
	t.Helper()
	line := strings.Repeat("REAL_DATA", 100) + "\n"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := f.WriteString(line); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
}

func fileSize(t *testing.T, p string) int64 {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat %s: %v", p, err)
	}
	return fi.Size()
}

func backupFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if e.Name() != "app.log" && strings.HasPrefix(e.Name(), "app-") {
			out = append(out, e.Name())
		}
	}
	return out
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
