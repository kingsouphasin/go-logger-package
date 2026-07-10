package httplogger

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logger "github.com/kingsouphasin/go-logger-package"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installLogger builds a console-off/file-off logger with the given body
// settings and installs it as the default, so the middleware's
// logger.BodyConfig() and logger.With() see it.
func installLogger(t *testing.T, logBody bool, maxBytes int) {
	t.Helper()
	l, err := logger.New(logger.Config{
		Env:          "production",
		Level:        "info",
		Console:      false,
		File:         false,
		LogBody:      logBody,
		MaxBodyBytes: maxBytes,
	})
	require.NoError(t, err)
	logger.SetDefault(l)
}

// TestHandlerReceivesFullBody is the correctness-critical case: even with body
// logging enabled (which reads r.Body), the handler must still receive the full
// original request body.
func TestHandlerReceivesFullBody(t *testing.T) {
	installLogger(t, true, 8)

	sent := `{"user":"alice","password":"secret123","note":"hello world"}`
	var got string
	h := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader(sent))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, sent, got, "handler must receive the full, unmodified request body")
}

// TestBodyDisabledStillWorks ensures the default (LogBody=false) path leaves the
// request body intact and does not error.
func TestBodyDisabledStillWorks(t *testing.T) {
	installLogger(t, false, 4096)

	sent := `{"a":1}`
	var got string
	h := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader(sent))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, sent, got)
}

// TestResponseWriterCapturesUpToLimit verifies the response buffer stops at the
// configured limit (+1) so large responses do not blow up memory, while the
// size counter still reflects the full write.
func TestResponseWriterCapturesUpToLimit(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), bodyLimit: 10}
	_, _ = rw.Write([]byte(strings.Repeat("y", 100)))

	assert.LessOrEqual(t, rw.body.Len(), 11, "capture must be bounded by bodyLimit+1")
	assert.Equal(t, 100, rw.size, "size counter must reflect the full write")
}
