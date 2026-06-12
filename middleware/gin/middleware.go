package ginlogger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		log := logger.With(
			zap.String("method", c.Request.Method),
			zap.String("path", path),
		)
		ctx := logger.WithContext(c.Request.Context(), log)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		log.Info("request completed",
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
