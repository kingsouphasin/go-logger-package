package chilogger

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		rw.Header().Set("X-Request-ID", requestID)

		contentType := r.Header.Get("Content-Type")
		log := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("query", r.URL.RawQuery),
			zap.String("ip", clientIP(r)),
			zap.String("user_agent", r.UserAgent()),
			zap.String("content_type", contentType),
			zap.Int64("request_size", r.ContentLength),
		)

		if strings.Contains(contentType, "multipart/form-data") {
			if files := fileUploads(r); len(files) > 0 {
				log = log.With(zap.Any("uploaded_files", files))
			}
		}

		ctx := logger.WithContext(r.Context(), log)
		next.ServeHTTP(rw, r.WithContext(ctx))

		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}

		log.Info("request completed",
			zap.String("path", path),
			zap.Int("status", rw.statusCode),
			zap.Int("response_size", rw.size),
			zap.Duration("latency", time.Since(start)),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
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

type uploadedFile struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.SplitN(ip, ",", 2)[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func fileUploads(r *http.Request) []uploadedFile {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil
	}
	var files []uploadedFile
	for _, headers := range r.MultipartForm.File {
		for _, h := range headers {
			ct := h.Header.Get("Content-Type")
			if ct == "" {
				ct = "application/octet-stream"
			}
			files = append(files, uploadedFile{
				Name:        h.Filename,
				Size:        h.Size,
				ContentType: ct,
			})
		}
	}
	return files
}
