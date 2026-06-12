package chilogger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		log := logger.With(
			zap.String("method", r.Method),
		)
		ctx := logger.WithContext(r.Context(), log)

		next.ServeHTTP(rw, r.WithContext(ctx))

		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}

		log.Info("request completed",
			zap.String("path", path),
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
