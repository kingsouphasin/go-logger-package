package fiberlogger

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

const loggerKey = "logger"

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
			zap.String("query", string(c.Request().URI().QueryString())),
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
