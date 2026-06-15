package fiberlogger

import (
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kingsouphasin/go-logger-package"
	"go.uber.org/zap"
)

const loggerKey = "logger"

// Handle wraps a handler function that accepts a logger, so callers do not need
// to call FromFiberCtx themselves. The middleware must be registered first.
func Handle(fn func(*fiber.Ctx, logger.Logger) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return fn(c, FromFiberCtx(c))
	}
}

func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)

		contentType := string(c.Request().Header.ContentType())
		log := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("query", sanitizeQuery(string(c.Request().URI().QueryString()))),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
			zap.String("content_type", contentType),
			zap.Int("request_size", c.Request().Header.ContentLength()),
		)

		if strings.Contains(contentType, "multipart/form-data") {
			if files := fileUploads(c); len(files) > 0 {
				log = log.With(zap.Any("uploaded_files", files))
			}
		}

		c.Locals(loggerKey, log)
		err := c.Next()

		log.Info("request completed",
			zap.Int("status", c.Response().StatusCode()),
			zap.Int("response_size", len(c.Response().Body())),
			zap.Duration("latency", time.Since(start)),
		)
		return err
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

func fileUploads(c *fiber.Ctx) []uploadedFile {
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

// FromFiberCtx retrieves the logger injected by Middleware from Fiber locals.
// Falls back to the logger stored in the request's context (or the global default).
func FromFiberCtx(c *fiber.Ctx) logger.Logger {
	if l, ok := c.Locals(loggerKey).(logger.Logger); ok {
		return l
	}
	return logger.FromContext(c.Context())
}
