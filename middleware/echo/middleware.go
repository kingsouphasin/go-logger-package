package echologger

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kingsouphasin/logger"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Response().Header().Set("X-Request-ID", requestID)

			contentType := c.Request().Header.Get("Content-Type")
			log := logger.With(
				zap.String("request_id", requestID),
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.String("query", c.Request().URL.RawQuery),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.String("content_type", contentType),
				zap.Int64("request_size", c.Request().ContentLength),
			)

			if strings.Contains(contentType, "multipart/form-data") {
				if files := fileUploads(c); len(files) > 0 {
					log = log.With(zap.Any("uploaded_files", files))
				}
			}

			ctx := logger.WithContext(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			status := c.Response().Status
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			}

			log.Info("request completed",
				zap.Int("status", status),
				zap.Int64("response_size", c.Response().Size),
				zap.Duration("latency", time.Since(start)),
			)
			return err
		}
	}
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
