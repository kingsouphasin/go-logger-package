package echologger

import (
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

			log := logger.With(
				zap.String("request_id", requestID),
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
			)
			ctx := logger.WithContext(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			status := c.Response().Status
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			}

			log.Info("request completed",
				zap.Int("status", status),
				zap.Duration("latency", time.Since(start)),
			)
			return err
		}
	}
}
