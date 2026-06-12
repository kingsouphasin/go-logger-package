package chilogger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kingsouphasin/logger"
	"github.com/stretchr/testify/assert"
)

func TestChiMiddlewareInjectsLogger(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Middleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		assert.NotNil(t, log)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestChiMiddlewareCapturesStatus(t *testing.T) {
	r := chi.NewRouter()
	r.Use(Middleware)
	r.Get("/created", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/created", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}
