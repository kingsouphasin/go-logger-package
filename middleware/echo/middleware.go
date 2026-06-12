package echologger

import (
	"time"

	"github.com/kingsouphasin/logger"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			log := logger.With(
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
			)
			ctx := logger.WithContext(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			log.Info("request completed",
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", time.Since(start)),
			)
			return err
		}
	}
}
