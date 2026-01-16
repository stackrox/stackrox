package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imageintegration"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	intergrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	scannerTypesMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestScanSBOMHttpHandler_ServeHTTP(t *testing.T) {
	t.Run("scanner v4 disabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, false)

		req := httptest.NewRequest(http.MethodGet, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMScanHandler(imageintegration.Set())
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
		assert.Contains(t, string(body), "Scanner V4 is disabled")
	})

	t.Run("invalid request method", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, true)

		req := httptest.NewRequest(http.MethodGet, "/sbom", nil)
		recorder := httptest.NewRecorder()

		handler := SBOMScanHandler(imageintegration.Set())
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
		assert.Contains(t, string(body), "SBOM Scanning is disabled")
	})

	t.Run("invalid media type", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, true)
		testutils.MustUpdateFeature(t, features.SBOMScanning, true)

		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		req.Header.Add("Content-Type", "wrong")
		recorder := httptest.NewRecorder()

		handler := SBOMScanHandler(imageintegration.Set())
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("scanner v4 integration missing", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, true)
		testutils.MustUpdateFeature(t, features.SBOMScanning, true)

		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		req.Header.Add("Content-Type", supportedMediaTypes.AsSlice()[0])
		recorder := httptest.NewRecorder()

		handler := SBOMScanHandler(imageintegration.Set())
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Contains(t, string(body), "integration")
	})

	t.Run("scanner v4 scan error", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, true)
		testutils.MustUpdateFeature(t, features.SBOMScanning, true)

		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		req.Header.Add("Content-Type", supportedMediaTypes.AsSlice()[0])
		recorder := httptest.NewRecorder()

		ctrl := gomock.NewController(t)

		mockSBOMScanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		mockSBOMScanner.EXPECT().ScanSBOM(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("fake error"))
		mockSBOMScanner.EXPECT().Type().Return(scannerTypes.ScannerV4)

		mockImageScannerWithDS := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		mockImageScannerWithDS.EXPECT().GetScanner().Return(mockSBOMScanner).AnyTimes()
		mockImageScannerWithDS.EXPECT().DataSource().Return(&storage.DataSource{}).AnyTimes()

		mockScannerSet := scannerMocks.NewMockSet(ctrl)
		mockScannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{mockImageScannerWithDS})

		mockIntegrationSet := intergrationMocks.NewMockSet(ctrl)
		mockIntegrationSet.EXPECT().ScannerSet().Return(mockScannerSet)

		handler := SBOMScanHandler(mockIntegrationSet)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Contains(t, string(body), "scanning sbom")
	})

	t.Run("valid scan", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.ScannerV4, true)
		testutils.MustUpdateFeature(t, features.SBOMScanning, true)

		req := httptest.NewRequest(http.MethodPost, "/sbom", nil)
		req.Header.Add("Content-Type", supportedMediaTypes.AsSlice()[0])
		recorder := httptest.NewRecorder()

		ctrl := gomock.NewController(t)

		mockSBOMScanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		mockSBOMScanner.EXPECT().ScanSBOM(gomock.Any(), gomock.Any(), gomock.Any()).Return(&v1.SBOMScanResponse{Id: "fake-sbom-id"}, nil)
		mockSBOMScanner.EXPECT().Type().Return(scannerTypes.ScannerV4)

		mockImageScannerWithDS := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		mockImageScannerWithDS.EXPECT().GetScanner().Return(mockSBOMScanner).AnyTimes()
		mockImageScannerWithDS.EXPECT().DataSource().Return(&storage.DataSource{}).AnyTimes()

		mockScannerSet := scannerMocks.NewMockSet(ctrl)
		mockScannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{mockImageScannerWithDS})

		mockIntegrationSet := intergrationMocks.NewMockSet(ctrl)
		mockIntegrationSet.EXPECT().ScannerSet().Return(mockScannerSet)

		handler := SBOMScanHandler(mockIntegrationSet)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "fake-sbom-id")
	})

}
