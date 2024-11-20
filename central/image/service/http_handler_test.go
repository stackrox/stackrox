package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	imageService "github.com/stackrox/rox/central/image/service"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stretchr/testify/assert"
)

func TestHttpHandler_ServeHTTP(t *testing.T) {

	// Test case: Invalid request method
	t.Run("Invalid request method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := imageService.Handler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		// Assert
		res := recorder.Result()
		defer res.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	})

}
