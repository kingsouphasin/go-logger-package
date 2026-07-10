package echologger

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kingsouphasin/go-logger-package"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Handle wraps a handler function that accepts a logger, so callers do not need
// to call logger.FromContext themselves. The middleware must be registered first.
func Handle(fn func(echo.Context, logger.Logger) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		return fn(c, logger.FromContext(c.Request().Context()))
	}
}

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Response().Header().Set("X-Request-ID", requestID)

			bodyEnabled, bodyMax := logger.BodyConfig()

			contentType := c.Request().Header.Get("Content-Type")
			log := logger.With(
				zap.String("request_id", requestID),
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.String("query", sanitizeQuery(c.Request().URL.RawQuery)),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.String("content_type", contentType),
				zap.Int64("request_size", c.Request().ContentLength),
			)

			var reqBody string
			if strings.Contains(contentType, "multipart/form-data") {
				if files := fileUploads(c); len(files) > 0 {
					log = log.With(zap.Any("uploaded_files", files))
				}
			} else if bodyEnabled && isBodyLoggable(contentType) {
				if raw, err := io.ReadAll(c.Request().Body); err == nil {
					c.Request().Body = io.NopCloser(bytes.NewReader(raw)) // restore for the handler
					reqBody = captureBody(raw, bodyMax, contentType)
				}
			}

			var bw *bodyWriter
			if bodyEnabled {
				bw = &bodyWriter{ResponseWriter: c.Response().Writer, bodyLimit: bodyMax}
				c.Response().Writer = bw
			}

			ctx := logger.WithContext(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			mw := log.WithoutCaller()

			if reqBody != "" {
				mw.Info("HTTP Request", zap.String("request_body", reqBody))
			} else {
				mw.Info("HTTP Request")
			}

			err := next(c)

			status := c.Response().Status
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			}

			respFields := []zap.Field{
				zap.String("route", c.Path()),
				zap.Int("status", status),
				zap.Int64("response_size", c.Response().Size),
				zap.String("latency", time.Since(start).String()),
			}
			if bw != nil && isBodyLoggable(c.Response().Header().Get("Content-Type")) {
				if b := captureBody(bw.body.Bytes(), bodyMax, c.Response().Header().Get("Content-Type")); b != "" {
					respFields = append(respFields, zap.String("response_body", b))
				}
			}
			mw.Info("HTTP Response", respFields...)
			return err
		}
	}
}

// bodyWriter wraps the echo response writer to capture up to bodyLimit+1 bytes
// of the response body while passing writes through unchanged.
type bodyWriter struct {
	http.ResponseWriter
	body      bytes.Buffer
	bodyLimit int
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	if w.bodyLimit > 0 && w.body.Len() <= w.bodyLimit {
		if remaining := w.bodyLimit + 1 - w.body.Len(); len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
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

func fileUploads(c echo.Context) []uploadedFile {
	form, err := c.MultipartForm()
	if err != nil {
		return nil
	}
	var files []uploadedFile
	for _, headers := range form.File {
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
