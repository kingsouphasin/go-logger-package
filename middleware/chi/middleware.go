package chilogger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kingsouphasin/go-logger-package"
	"go.uber.org/zap"
)

// Handle wraps a handler function that accepts a logger, so callers do not need
// to call logger.FromContext themselves. The middleware must be registered first.
func Handle(fn func(http.ResponseWriter, *http.Request, logger.Logger)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, logger.FromContext(r.Context()))
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		rw.Header().Set("X-Request-ID", requestID)

		bodyEnabled, bodyMax := logger.BodyConfig()
		if bodyEnabled {
			rw.bodyLimit = bodyMax
		}

		contentType := r.Header.Get("Content-Type")
		log := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", sanitizeQuery(r.URL.RawQuery)),
			zap.String("ip", clientIP(r)),
			zap.String("user_agent", r.UserAgent()),
			zap.String("content_type", contentType),
			zap.Int64("request_size", r.ContentLength),
		)

		var reqBody string
		if strings.Contains(contentType, "multipart/form-data") {
			if files := fileUploads(r); len(files) > 0 {
				log = log.With(zap.Any("uploaded_files", files))
			}
		} else if bodyEnabled && isBodyLoggable(contentType) {
			if raw, err := io.ReadAll(r.Body); err == nil {
				r.Body = io.NopCloser(bytes.NewReader(raw)) // restore for the handler
				reqBody = captureBody(raw, bodyMax, contentType)
			}
		}

		ctx := logger.WithContext(r.Context(), log)
		mw := log.WithoutCaller()

		if reqBody != "" {
			mw.Info("HTTP Request", zap.String("request_body", reqBody))
		} else {
			mw.Info("HTTP Request")
		}

		next.ServeHTTP(rw, r.WithContext(ctx))

		route := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			route = rctx.RoutePattern()
		}

		respFields := []zap.Field{
			zap.String("route", route),
			zap.Int("status", rw.statusCode),
			zap.Int("response_size", rw.size),
			zap.String("latency", time.Since(start).String()),
		}
		if bodyEnabled && isBodyLoggable(rw.Header().Get("Content-Type")) {
			if b := captureBody(rw.body.Bytes(), bodyMax, rw.Header().Get("Content-Type")); b != "" {
				respFields = append(respFields, zap.String("response_body", b))
			}
		}
		mw.Info("HTTP Response", respFields...)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	body       bytes.Buffer // captured response body (empty unless bodyLimit > 0)
	bodyLimit  int          // max bytes to capture; 0 disables capture
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	if rw.bodyLimit > 0 && rw.body.Len() <= rw.bodyLimit {
		if remaining := rw.bodyLimit + 1 - rw.body.Len(); len(b) > remaining {
			rw.body.Write(b[:remaining])
		} else {
			rw.body.Write(b)
		}
	}
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

var sensitiveQueryKeys = map[string]struct{}{
	"token": {}, "access_token": {}, "api_key": {}, "key": {},
	"password": {}, "code": {}, "state": {}, "authorization": {},
	"secret": {}, "client_secret": {},
}

func sanitizeQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "[unparseable]"
	}
	for k := range q {
		if _, sensitive := sensitiveQueryKeys[strings.ToLower(k)]; sensitive {
			q[k] = []string{"[redacted]"}
		}
	}
	return q.Encode()
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
