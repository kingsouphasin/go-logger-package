package chilogger

import (
	"net/http"
	"time"

	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware(next http.Handler) http.Handler {
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

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
