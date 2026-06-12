package fiberlogger

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kingsouphasin/logger"
	"go.uber.org/zap"
)

const loggerKey = "logger"

func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		log := logger.With(
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
		)
		c.Locals(loggerKey, log)

		err := c.Next()

		log.Info("request completed",
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
		)
		return err
	}
}

// FromFiberCtx retrieves the logger injected by Middleware from Fiber locals.
// Falls back to the logger stored in the request's context (or the global default).
func FromFiberCtx(c *fiber.Ctx) logger.Logger {
	if l, ok := c.Locals(loggerKey).(logger.Logger); ok {
		return l
	}
	return logger.FromContext(c.Context())
}
