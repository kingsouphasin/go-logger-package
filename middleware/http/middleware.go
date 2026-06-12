package httplogger

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			log := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			ctx := logger.WithContext(r.Context(), log)

			next.ServeHTTP(rw, r.WithContext(ctx))

			log.Info("request completed",
				zap.Int("status", rw.statusCode),
				zap.Duration("latency", time.Since(start)),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("responseWriter does not implement http.Hijacker")
}
