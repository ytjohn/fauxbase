package testutil

import (
	"net/http"
	"net/http/httptest"

	"github.com/ytjohn/fauxbase/pkg/server/middleware"
)

// CreateTestResponseWriter creates a new mock response writer
func CreateTestResponseWriter() *middleware.ResponseWriter {
	recorder := httptest.NewRecorder()
	return &middleware.ResponseWriter{
		ResponseWriter: recorder,
		Status:         http.StatusOK,
	}
}
