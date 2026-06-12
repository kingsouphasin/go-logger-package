package echologger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kingsouphasin/logger"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEchoMiddlewareInjectsLogger(t *testing.T) {
	e := echo.New()
	e.Use(Middleware())
	e.GET("/test", func(c echo.Context) error {
		log := logger.FromContext(c.Request().Context())
		assert.NotNil(t, log)
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestEchoMiddlewareCaptures500(t *testing.T) {
	e := echo.New()
	e.Use(Middleware())
	e.GET("/error", func(c echo.Context) error {
		return c.NoContent(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
