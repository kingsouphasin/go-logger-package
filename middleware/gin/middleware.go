package ginlogger

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)

		contentType := c.ContentType()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		log := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", c.Request.URL.RawQuery),
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

		c.Next()

		log.Info("request completed",
			zap.Int("status", c.Writer.Status()),
			zap.Int("response_size", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
		)
	}
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
