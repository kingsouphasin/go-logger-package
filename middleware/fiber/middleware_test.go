package fiberlogger

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiberMiddlewareInjectsLogger(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		log := FromFiberCtx(c)
		assert.NotNil(t, log)
		return c.SendStatus(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFiberMiddlewareFromFiberCtxFallsBackToDefault(t *testing.T) {
	app := fiber.New()
	app.Get("/no-middleware", func(c *fiber.Ctx) error {
		log := FromFiberCtx(c)
		assert.NotNil(t, log)
		assert.Equal(t, logger.FromContext(c.Context()), log)
		return c.SendStatus(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/no-middleware", nil)
	require.NoError(t, err)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
