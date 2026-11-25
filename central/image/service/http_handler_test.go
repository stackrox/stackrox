package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	"github.com/stackrox/rox/central/imageintegration"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	imageV2Mocks "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	enricherMock "github.com/stackrox/rox/pkg/images/enricher/mocks"
	intergrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	scannerTypesMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		handler := SBOMGenHandler(imageintegration.Set(), nil, nil, nil, nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	})

	// Test case: valid json body and enricher returns error
	t.Run("valid json body with error from enricher", func(t *testing.T) {
		testcases := []struct {
			name                    string
			flattenImageDataEnabled bool
		}{
			{name: "with flatten image data disabled", flattenImageDataEnabled: false},
			{name: "with flatten image data enabled", flattenImageDataEnabled: true},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				testutils.MustUpdateFeature(t, features.FlattenImageData, tc.flattenImageDataEnabled)

				t.Setenv(features.ScannerV4.EnvVar(), "true")

				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				// initialize mocks
				mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
				mockEnricherV2 := enricherMock.NewMockImageEnricherV2(ctrl)
				// Without a Scanner V4 integration the handler fails before reaching the
				// enricher, so provide one to actually exercise the enrichment error path.
				set := intergrationMocks.NewMockSet(ctrl)
				scannerSet := scannerMocks.NewMockSet(ctrl)
				scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
				integration := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
				set.EXPECT().ScannerSet().Return(scannerSet)
				scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{integration})
				integration.EXPECT().GetScanner().Return(scanner).AnyTimes()
				integration.EXPECT().DataSource().Return(nil)
				scanner.EXPECT().Type().Return(scannerTypes.ScannerV4)

				// Exactly the enricher selected by the feature flag must be called once.
				enrichErr := errors.New("Image enrichment failed")
				if tc.flattenImageDataEnabled {
					mockEnricherV2.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanNotDone}, enrichErr).Times(1)
				} else {
					mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanNotDone}, enrichErr).Times(1)
				}

				reqBody := &apiparams.SBOMRequestBody{
					ImageName: "test-image",
					Force:     false,
				}
				reqJson, err := json.Marshal(reqBody)
				assert.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
				recorder := httptest.NewRecorder()

				handler := SBOMGenHandler(set, mockEnricher, mockEnricherV2, nil, nil, nil, nil)
				handler.ServeHTTP(recorder, req)

				res := recorder.Result()
				err = res.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
				assert.Contains(t, recorder.Body.String(), "Image enrichment failed")
			})
		}
	})

	// Test case: valid json body and validate enricher being called
	t.Run("valid json body", func(t *testing.T) {
		testcases := []struct {
			name                    string
			flattenImageDataEnabled bool
			scanResult              enricher.ScanResult
			imageUpdated            bool
		}{
			{name: "with flatten image data disabled", flattenImageDataEnabled: false, scanResult: enricher.ScanSucceeded, imageUpdated: true},
			{name: "with flatten image data enabled", flattenImageDataEnabled: true, scanResult: enricher.ScanSucceeded, imageUpdated: true},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				testutils.MustUpdateFeature(t, features.FlattenImageData, tc.flattenImageDataEnabled)
				t.Setenv(features.ScannerV4.EnvVar(), "true")

				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// initialize mocks
				scannerSet := scannerMocks.NewMockSet(ctrl)
				set := intergrationMocks.NewMockSet(ctrl)
				mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
				mockEnricherV2 := enricherMock.NewMockImageEnricherV2(ctrl)
				scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
				fsr := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)

				mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: tc.imageUpdated, ScanResult: tc.scanResult}, nil).AnyTimes()
				mockEnricherV2.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).Return(enricher.EnrichmentResult{ImageUpdated: tc.imageUpdated, ScanResult: tc.scanResult}, nil).AnyTimes()
				scanner.EXPECT().Type().Return(scannerTypes.ScannerV4).AnyTimes()
				scanner.EXPECT().GetSBOM(gomock.Any()).DoAndReturn(getFakeSBOM).AnyTimes()
				set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()
				fsr.EXPECT().GetScanner().Return(scanner).AnyTimes()
				fsr.EXPECT().DataSource().Return(nil).AnyTimes()
				scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{fsr}).AnyTimes()

				reqBody := &apiparams.SBOMRequestBody{
					ImageName: "quay.io/quay-qetest/nodejs-test-image:latest",
					Force:     false,
				}

				reqJson, err := json.Marshal(reqBody)
				assert.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqJson))
				recorder := httptest.NewRecorder()

				handler := SBOMGenHandler(set, mockEnricher, mockEnricherV2, nil, nil, nil, nil)
				handler.ServeHTTP(recorder, req)

				res := recorder.Result()
				err = res.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, res.StatusCode)
			})
		}
	})

	// Test case: invalid json body
	t.Run("invalid json body", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		invalidJson := `{"cluster": "test-cluster", "imageName": "test-image", "force": true,`
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewBufferString(invalidJson))
		recorder := httptest.NewRecorder()

		handler := SBOMGenHandler(imageintegration.Set(), nil, nil, nil, nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	// Test case: empty image name
	t.Run("empty image name", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "true")

		reqBody := []byte(`{"imageName": "   "}`)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(reqBody))
		recorder := httptest.NewRecorder()

		handler := SBOMGenHandler(imageintegration.Set(), nil, nil, nil, nil, nil, nil)
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

		handler := SBOMGenHandler(imageintegration.Set(), nil, nil, nil, nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
	})

	// Test case: request body size exceeds limit
	t.Run("request body size exceeds limit", func(t *testing.T) {
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		t.Setenv(env.SBOMGenerationMaxReqSizeBytes.EnvVar(), "2")
		largeRequestBody := []byte(`{"cluster": "test-cluster", "imageName": "test-image", "force": true}`)
		req := httptest.NewRequest(http.MethodPost, "/sbom", bytes.NewReader(largeRequestBody))
		recorder := httptest.NewRecorder()

		handler := SBOMGenHandler(imageintegration.Set(), nil, nil, nil, nil, nil, nil)
		handler.ServeHTTP(recorder, req)

		res := recorder.Result()
		err := res.Body.Close()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}

func TestSBOMCachedImage(t *testing.T) {
	const imageName = "registry.example.org/app:latest"
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, v2 := range []bool{false, true} {
		for name, tc := range map[string]struct{ readable, indexFound, force bool }{
			"cache hit":            {readable: true, indexFound: true},
			"missing index report": {readable: true},
			"outside read scope":   {},
			"explicit force":       {readable: true, force: true},
		} {
			t.Run(fmt.Sprintf("v2=%t/%s", v2, name), func(t *testing.T) {
				testutils.MustUpdateFeature(t, features.FlattenImageData, v2)
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				ctrl := gomock.NewController(t)
				images := imageMocks.NewMockDataStore(ctrl)
				imagesV2 := imageV2Mocks.NewMockDataStore(ctrl)
				e := enricherMock.NewMockImageEnricher(ctrl)
				eV2 := enricherMock.NewMockImageEnricherV2(ctrl)
				risk := riskManagerMocks.NewMockManager(ctrl)
				set := intergrationMocks.NewMockSet(ctrl)
				scannerSet := scannerMocks.NewMockSet(ctrl)
				scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
				integration := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
				set.EXPECT().ScannerSet().Return(scannerSet)
				scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{integration})
				integration.EXPECT().GetScanner().Return(scanner).AnyTimes()
				integration.EXPECT().DataSource().Return(nil)
				scanner.EXPECT().Type().Return(scannerTypes.ScannerV4)

				img := &storage.Image{
					Id: digest, Name: &storage.ImageName{FullName: imageName}, IsClusterLocal: true,
					Metadata: &storage.ImageMetadata{},
					Scan:     &storage.ImageScan{DataSource: &storage.DataSource{Id: iiStore.DefaultScannerV4Integration.GetId()}},
				}
				ctx := sac.WithNoAccess(t.Context())
				if tc.readable {
					ctx = sac.WithAllAccess(t.Context())
				}
				if !tc.force {
					callerScope := gomock.Cond(func(ctx context.Context) bool {
						allowed, err := sac.ForResource(resources.Image).ReadAllowed(ctx)
						return err == nil && allowed == tc.readable
					})
					if v2 {
						var stored *storage.ImageV2
						if tc.readable {
							stored = utils.ConvertToV2(img)
						}
						imagesV2.EXPECT().GetImage(callerScope, utils.NewImageV2ID(img.Name, digest)).Return(stored, tc.readable, nil)
					} else {
						var stored *storage.Image
						if tc.readable {
							stored = img
						}
						images.EXPECT().GetImage(callerScope, digest).Return(stored, tc.readable, nil)
					}
					if tc.readable {
						scanner.EXPECT().GetSBOM(gomock.Cond(func(i *storage.Image) bool { return i.GetIsClusterLocal() && i.GetId() == digest })).Return([]byte(`{"cached":true}`), tc.indexFound, nil)
					}
				}
				if !tc.indexFound {
					fresh := gomock.Cond(func(ec enricher.EnrichmentContext) bool {
						return ec.FetchOpt == enricher.UseImageNamesRefetchCachedValues && ec.ScannerTypeHint == scannerTypes.ScannerV4
					})
					result := enricher.EnrichmentResult{ScanResult: enricher.ScanSucceeded}
					var err error
					if !tc.readable {
						err = errors.New("registry unavailable")
					}
					if v2 {
						eV2.EXPECT().EnrichImage(gomock.Any(), fresh, gomock.Any()).Return(result, err)
						if tc.readable {
							risk.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Any())
						}
					} else {
						e.EXPECT().EnrichImage(gomock.Any(), fresh, gomock.Any()).Return(result, err)
						if tc.readable {
							risk.EXPECT().CalculateRiskAndUpsertImage(gomock.Any())
						}
					}
					if tc.readable {
						scanner.EXPECT().GetSBOM(gomock.Any()).Return([]byte(`{"cached":false}`), true, nil)
					}
				}
				body, err := json.Marshal(apiparams.SBOMRequestBody{ImageName: imageName, Digest: digest, Force: tc.force})
				require.NoError(t, err)
				req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/sbom", bytes.NewReader(body))
				w := httptest.NewRecorder()
				SBOMGenHandler(set, e, eV2, nil, risk, images, imagesV2).ServeHTTP(w, req)
				if !tc.readable {
					require.Equal(t, http.StatusInternalServerError, w.Code)
					require.Contains(t, w.Body.String(), "registry unavailable")
					return
				}
				require.Equal(t, http.StatusOK, w.Code)
				require.JSONEq(t, fmt.Sprintf(`{"cached":%t}`, tc.indexFound), w.Body.String())
			})
		}
	}
}

func TestSBOMCachedImageFallbacks(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// scannerV4Mocks returns a mock integration set yielding the given Scanner V4 SBOMer.
	scannerV4Mocks := func(ctrl *gomock.Controller, scanner *scannerTypesMocks.MockScannerSBOMer) *intergrationMocks.MockSet {
		set := intergrationMocks.NewMockSet(ctrl)
		scannerSet := scannerMocks.NewMockSet(ctrl)
		integration := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		set.EXPECT().ScannerSet().Return(scannerSet)
		scannerSet.EXPECT().GetAll().Return([]scannerTypes.ImageScannerWithDataSource{integration})
		integration.EXPECT().GetScanner().Return(scanner).AnyTimes()
		integration.EXPECT().DataSource().Return(nil)
		scanner.EXPECT().Type().Return(scannerTypes.ScannerV4)
		return set
	}

	serveSBOMRequest := func(handler http.Handler) *httptest.ResponseRecorder {
		reqJson, err := json.Marshal(apiparams.SBOMRequestBody{ImageName: "registry.example.org/app:latest", Digest: digest})
		require.NoError(t, err)
		req := httptest.NewRequestWithContext(sac.WithAllAccess(t.Context()), http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	t.Run("inline digest reference finds stored V2 row", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.FlattenImageData, true)
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		ctrl := gomock.NewController(t)

		img := &storage.Image{
			Id: digest, Name: &storage.ImageName{FullName: "registry.example.org/app:latest"}, IsClusterLocal: true,
			Metadata: &storage.ImageMetadata{},
			Scan:     &storage.ImageScan{DataSource: &storage.DataSource{Id: iiStore.DefaultScannerV4Integration.GetId()}},
		}

		// The with-suffix key misses; the stripped key finds the stored row.
		ci, err := containerImage(apiparams.SBOMRequestBody{ImageName: "registry.example.org/app:latest@" + digest})
		require.NoError(t, err)
		strippedName := ci.GetName().CloneVT()
		strippedName.FullName = "registry.example.org/app:latest"

		imagesV2 := imageV2Mocks.NewMockDataStore(ctrl)
		imagesV2.EXPECT().GetImage(gomock.Any(), ci.GetIdV2()).Return(nil, false, nil)
		imagesV2.EXPECT().GetImage(gomock.Any(), utils.NewImageV2ID(strippedName, digest)).
			Return(utils.ConvertToV2(img), true, nil)

		scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		scanner.EXPECT().GetSBOM(gomock.Any()).Return([]byte(`{"cached":true}`), true, nil)

		handler := SBOMGenHandler(scannerV4Mocks(ctrl, scanner), nil, nil, nil, nil, nil, imagesV2)
		reqJson, err := json.Marshal(apiparams.SBOMRequestBody{ImageName: "registry.example.org/app:latest@" + digest})
		require.NoError(t, err)
		req := httptest.NewRequestWithContext(sac.WithAllAccess(t.Context()), http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"cached":true}`, w.Body.String())
	})

	t.Run("cache read error falls back to enrichment", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.FlattenImageData, false)
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		ctrl := gomock.NewController(t)

		images := imageMocks.NewMockDataStore(ctrl)
		images.EXPECT().GetImage(gomock.Any(), digest).Return(nil, false, errors.New("db unavailable"))

		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(enricher.EnrichmentResult{ScanResult: enricher.ScanSucceeded}, nil).Times(1)

		risk := riskManagerMocks.NewMockManager(ctrl)
		risk.EXPECT().CalculateRiskAndUpsertImage(gomock.Any())

		scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		scanner.EXPECT().GetSBOM(gomock.Any()).Return([]byte(`{"fresh":true}`), true, nil)

		w := serveSBOMRequest(SBOMGenHandler(scannerV4Mocks(ctrl, scanner), mockEnricher, nil, nil, risk, images, nil))
		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"fresh":true}`, w.Body.String())
	})

	t.Run("tag only request skips the cache and enriches", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.FlattenImageData, false)
		t.Setenv(features.ScannerV4.EnvVar(), "true")
		ctrl := gomock.NewController(t)

		// A tag does not identify an immutable image, so no cache lookup may happen.
		mockEnricher := enricherMock.NewMockImageEnricher(ctrl)
		mockEnricher.EXPECT().EnrichImage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(enricher.EnrichmentResult{ScanResult: enricher.ScanSucceeded}, nil).Times(1)

		// The mock enricher leaves the image without metadata, so saveImageV1 has no SHA
		// to key on and skips the upsert; no risk manager call is expected.

		scanner := scannerTypesMocks.NewMockScannerSBOMer(ctrl)
		scanner.EXPECT().GetSBOM(gomock.Any()).Return([]byte(`{"fresh":true}`), true, nil)

		reqJson, err := json.Marshal(apiparams.SBOMRequestBody{ImageName: "registry.example.org/app:latest"})
		require.NoError(t, err)
		req := httptest.NewRequestWithContext(sac.WithAllAccess(t.Context()), http.MethodPost, "/sbom", bytes.NewReader(reqJson))
		w := httptest.NewRecorder()
		SBOMGenHandler(scannerV4Mocks(ctrl, scanner), mockEnricher, nil, nil, riskManagerMocks.NewMockManager(ctrl), nil, nil).ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"fresh":true}`, w.Body.String())
	})
}
