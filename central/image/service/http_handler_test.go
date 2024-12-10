package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	enricherMock "github.com/stackrox/rox/pkg/images/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHttpHandler_ServeHTTP(t *testing.T) {

	// Test case: Invalid request method
	t.Run("Invalid request method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	})

	// Test case: valid json body and enricher returns error
	t.Run("valid json body with error from enricher", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanNotDone}, errors.New("Image enrichment failed")).AnyTimes()

		reqBody := &sbomRequestBody{
			ImageName: "test-image",
			Force:     false,
		}
		reqJson, err := json.Marshal(reqBody)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), mockEnricher, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	})

	// Test case: valid json body and validate enricher being called
	t.Run("valid json body", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanSucceeded}, nil).AnyTimes()

		reqBody := &sbomRequestBody{
			ImageName: "test-image",
			Force:     false,
		}
		reqJson, err := json.Marshal(reqBody)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), mockEnricher, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	// Test case: invalid json body
	t.Run("invalid json body", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		invalidJson := `{"cluster": "test-cluster", "image_name": "test-image", "force": true,`
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewBufferString(invalidJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	// Test case: Scanner V4 not enabled
	t.Run("Scanner V4 not enabled", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "false")
		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
	})

	// Test case: SBOM feature not enabled
	t.Run("SBOM feature not enabled", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		t.Setenv(features.SBOMGeneration.EnvVar(), "false")
		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
	})

	// Test case: request body size exceeds limit
	t.Run("request body size exceeds limit", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		t.Setenv(env.SBOMGenerationMaxReqSizeBytes.EnvVar(), "2")
		largeRequestBody := []byte(`{"cluster": "test-cluster", "image_name": "test-image", "force": true}`)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(largeRequestBody))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

}
