// handler/health_handler_test.go
package handler

import (
	"go-bank-api/router"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_Integration(t *testing.T) {
	// Setup router. For this test, handlers can be nil.
	r := router.NewRouter(nil, nil, nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Execute
	r.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	expectedBody := `{"status":"API is healthy and running"}`
	assert.JSONEq(t, expectedBody, rr.Body.String())
}
