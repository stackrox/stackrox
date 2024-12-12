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
<<<<<<< HEAD
	"github.com/stackrox/rox/pkg/apiparams"
=======
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
>>>>>>> 660b15f188 (Add image enrichment to handler)
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	enricherMock "github.com/stackrox/rox/pkg/images/enricher/mocks"
	"github.com/stackrox/rox/pkg/images/integration/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	scannerTypeMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	scannerv4Mocks "github.com/stackrox/rox/pkg/scannerv4/client/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

var _ scannerTypes.Scanner = (*fakeScanner)(nil)
var _ scannerTypes.SBOMer = (*fakeScanner)(nil)

type fakeScanner struct {
	requestedScan bool
	notMatch      bool
}

func (*fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeScanner) GetSBOM(_ *storage.Image) ([]byte, error, bool) {
	return []byte{}, nil, true
}

func (f *fakeScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	f.requestedScan = true
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2020-1234",
					},
				},
			},
		},
	}, nil
}

func (f *fakeScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeScanner) Test() error {
	return nil
}

func (*fakeScanner) Type() string {
	return scannerTypes.ScannerV4
}

func (*fakeScanner) Name() string {
	return "name"
}

func (*fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

var (
	_ scannerTypes.ImageScannerWithDataSource = (*fakeRegistryScanner)(nil)
	_ types.ImageRegistry                     = (*fakeRegistryScanner)(nil)
)

type fakeRegistryScanner struct {
	scanner           *fakeScanner
	requestedMetadata bool
	notMatch          bool
}

type opts struct {
	requestedScan     bool
	requestedMetadata bool
	notMatch          bool
}

func newFakeRegistryScanner(opts opts) *fakeRegistryScanner {
	return &fakeRegistryScanner{
		scanner: &fakeScanner{
			requestedScan: opts.requestedScan,
			notMatch:      opts.notMatch,
		},
		requestedMetadata: opts.requestedMetadata,
		notMatch:          opts.notMatch,
	}
}

func (f *fakeRegistryScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	f.requestedMetadata = true
	return &storage.ImageMetadata{}, nil
}

func (f *fakeRegistryScanner) Config(_ context.Context) *types.Config {
	return nil
}

func (f *fakeRegistryScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeRegistryScanner) Test() error {
	return nil
}

func (*fakeRegistryScanner) Type() string {
	return "type"
}

func (*fakeRegistryScanner) Name() string {
	return "name"
}

func (*fakeRegistryScanner) HTTPClient() *http.Client {
	return nil
}

func (f *fakeRegistryScanner) GetScanner() scannerTypes.Scanner {
	return f.scanner
}

func (f *fakeRegistryScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{
		Id:   "id",
		Name: f.Name(),
	}
}

func (f *fakeRegistryScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   "id",
		Name: f.Name(),
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

		// mockSbom returns sample sbom
		sbom := map[string]interface{}{
			"SPDXID":      "SPDXRef-DOCUMENT",
			"spdxVersion": "SPDX-2.3",
			"creationInfo": map[string]interface{}{
				"created": "2023-08-30T04:40:16Z",
				"creators": []string{
					"Organization: Uchiha Cortez",
					"Tool: FOSSA v0.12.0",
				},
			},
		}
		sbomBytes, err := json.Marshal(sbom)
		assert.NoError(t, err)

		// initliaze mocks
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockSbom := scannerTypeMocks.NewMockSBOMer(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanNotDone}, errors.New("Image enrichment failed")).AnyTimes()
		mockSbom.EXPECT().GetSBOM(gomock.Any()).Return(sbomBytes, nil, true).AnyTimes()

		reqBody := &apiparams.SbomRequestBody{
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

		// mockSbom returns sample sbom
		sbom := map[string]interface{}{
			"SPDXID":      "SPDXRef-DOCUMENT",
			"spdxVersion": "SPDX-2.3",
			"creationInfo": map[string]interface{}{
				"created": "2023-08-30T04:40:16Z",
				"creators": []string{
					"Organization: Uchiha Cortez",
					"Tool: FOSSA v0.12.0",
				},
			},
		}
		sbomBytes, err := json.Marshal(sbom)
		assert.NoError(t, err)

		// initliaze mocks
		scannerSet := scannerMocks.NewMockSet(ctrl)
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockSbom := scannerv4Mocks.NewMockScanner(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanSucceeded}, nil).AnyTimes()

<<<<<<< HEAD
		reqBody := &apiparams.SbomRequestBody{
			ImageName: "test-image",
=======
		var _ scannerTypes.Scanner = (*fakeScanner)(nil)

		mockSbom.EXPECT().GetSBOM(gomock.Any(), gomock.Any()).Return(sbomBytes, nil, true).AnyTimes()
		set := mocks.NewMockSet(ctrl)
		fsr := newFakeRegistryScanner(opts{})
		scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{fsr}).AnyTimes()
		set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()
		reqBody := &sbomRequestBody{
			ImageName: "quay.io/quay-qetest/nodejs-test-image:latest",
>>>>>>> 660b15f188 (Add image enrichment to handler)
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

}
