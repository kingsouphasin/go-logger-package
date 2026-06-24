package ginlogger

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kingsouphasin/go-logger-package"
	"go.uber.org/zap"
)

// Handle wraps a handler function that accepts a logger, so callers do not need
// to call logger.FromContext themselves. The middleware must be registered first.
func Handle(fn func(*gin.Context, logger.Logger)) gin.HandlerFunc {
	return func(c *gin.Context) {
		fn(c, logger.FromContext(c.Request.Context()))
	}
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)

		contentType := c.ContentType()

		log := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", sanitizeQuery(c.Request.URL.RawQuery)),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("content_type", contentType),
			zap.Int64("request_size", c.Request.ContentLength),
		)

		if strings.Contains(contentType, "multipart/form-data") {
			if files := fileUploads(c); len(files) > 0 {
				log = log.With(zap.Any("uploaded_files", files))
			}
		}

		ctx := logger.WithContext(c.Request.Context(), log)
		c.Request = c.Request.WithContext(ctx)

		mw := log.WithoutCaller()
		mw.Info("HTTP Request")
		c.Next()

		mw.Info("HTTP Response",
			zap.String("route", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Int("response_size", c.Writer.Size()),
			zap.String("latency", time.Since(start).String()),
		)
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

func fileUploads(c *gin.Context) []uploadedFile {
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
