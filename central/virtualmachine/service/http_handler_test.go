package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imageintegration"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	enricherMock "github.com/stackrox/rox/pkg/images/enricher/mocks"
	intergrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	scannerTypesMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func getFakeSBOM(_ any) ([]byte, bool, error) {
	sbom := createMockSBOM()
	sbomBytes, err := json.Marshal(sbom)
	if err != nil {
		return nil, false, err
	}
	return sbomBytes, true, nil
}

func createMockSBOM() map[string]interface{} {
	return map[string]interface{}{
		"SPDXID":      "SPDXRef-DOCUMENT",
		"spdxVersion": "SPDX-2.3",
		"creationInfo": map[string]interface{}{
			"created": "2023-08-30T04:40:16Z",
			"creators": []string{
				"Organization: xyz",
				"Tool: FOSSA v0.12.0",
			},
		},
	}
}

func TestHttpHandler_ServeHTTP(t *testing.T) {
	// Test case: Invalid request method
	t.Run("Invalid request method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
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
		// initialize mocks
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanNotDone}, errors.New("Image enrichment failed")).AnyTimes()

		reqBody := &apiparams.SBOMRequestBody{
			ImageName: "test-image",
			Force:     false,
		}
		reqJson, err := json.Marshal(reqBody)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), mockEnricher, nil, nil)
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

		// initialize mocks
		scannerSet := scannerMocks.NewMockSet(ctrl)
		set := intergrationMocks.NewMockSet(ctrl)
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		fsr := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)

		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanSucceeded}, nil).AnyTimes()
		scanner.EXPECT().Type().Return(scannerTypes.ScannerV4).AnyTimes()
		scanner.EXPECT().GetSBOM(gomock.Any()).DoAndReturn(getFakeSBOM).AnyTimes()
		set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()
		fsr.EXPECT().GetScanner().Return(scanner).AnyTimes()
		scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{fsr}).AnyTimes()

		reqBody := &apiparams.SBOMRequestBody{
			ImageName: "quay.io/quay-qetest/nodejs-test-image:latest",
			Force:     false,
		}

		reqJson, err := json.Marshal(reqBody)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(set, mockEnricher, nil, nil)
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
		invalidJson := `{"cluster": "test-cluster", "imageName": "test-image", "force": true,`
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewBufferString(invalidJson))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	// Test case: empty image name
	t.Run("empty image name", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")

		reqBody := []byte(`{"imageName": "   "}`)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqBody))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Contains(t, recorder.Body.String(), "image name")
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	// Test case: Scanner V4 not enabled
	t.Run("Scanner V4 not enabled", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "false")
		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
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

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
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
		largeRequestBody := []byte(`{"cluster": "test-cluster", "imageName": "test-image", "force": true}`)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(largeRequestBody))
		recorder := httptest.NewRecorder()

		handler := SBOMHandler(imageintegration.Set(), nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	// Tests case: edge case where an image scan existed in Central DB from Scanner V4
	// but the Scanner V4 Indexer did not have the corresponding index report.
	// A forced enrichment is expected and the result saved to Central DB.
	t.Run("image saved to Central when index report did not exist", func(t *testing.T) {
		t.Setenv(features.SBOMGeneration.EnvVar(), "true")
		t.Setenv(features.ScannerV4.EnvVar(), "true")

		// enrichImageFunc will fake enrich an image by Scanner V4.
		enrichImageFunc := func(ctx context.Context, enrichCtx enricher.EnrichmentContext, img *storage.Image) (enricher.EnrichmentResult, error) {
			img.Id = "fake-id"
			img.Scan = &storage.ImageScan{DataSource: &storage.DataSource{Id: iiStore.DefaultScannerV4Integration.Id}}
			return enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanSucceeded}, nil
		}

		// getSBOMFunc will indicate an index report is missing on first invocation and found on subsequent invocations.
		firstGetSBOMInvocation := true
		getSBOMFunc := func(_ any) ([]byte, bool, error) {
			if firstGetSBOMInvocation {
				firstGetSBOMInvocation = false
				return nil, false, nil
			}
			return getFakeSBOM(nil)
		}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Initialize mocks.
		mockScanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		mockScanner.EXPECT().GetSBOM(gomock.Any()).DoAndReturn(getSBOMFunc).AnyTimes()
		mockScanner.EXPECT().Type().Return(scannerTypes.ScannerV4).AnyTimes()

		mockImageScannerWithDS := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		mockImageScannerWithDS.EXPECT().GetScanner().Return(mockScanner).AnyTimes()

		mockScannerSet := scannerMocks.NewMockSet(ctrl)
		mockScannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{mockImageScannerWithDS}).AnyTimes()

		mockIntegrationSet := intergrationMocks.NewMockSet(ctrl)
		mockIntegrationSet.EXPECT().ScannerSet().Return(mockScannerSet).AnyTimes()

		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(enrichImageFunc).AnyTimes()

		mockRiskManager := riskManagerMocks.NewMockManager(ctrl)
		// Image is expected to be saved after each successful enrichment, for this edge case will
		// be multiple successful enrichments.
		mockRiskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Any()).Times(2)

		// Prepare the SBOM generation request.
		reqBody := &apiparams.SBOMRequestBody{
			ImageName: "quay.io/quay-qetest/nodejs-test-image:latest",
			Force:     false,
		}
		reqJson, err := json.Marshal(reqBody)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		recorder := httptest.NewRecorder()
		handler := SBOMHandler(mockIntegrationSet, mockEnricher, nil, mockRiskManager)

		// Make the SBOM generation request.
		handler.ServeHTTP(recorder, req)

		// Validate was successful.
		res := recorder.Result()
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
