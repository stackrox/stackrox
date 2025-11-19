package manager

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	imageV2DSMocks "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	evaluatorMocks "github.com/stackrox/rox/central/processbaseline/evaluator/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/risk/getters"
	deploymentScorer "github.com/stackrox/rox/central/risk/scorer/deployment"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	integrationSetMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	scannerSetMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/scanners/types"
	scannerTypesMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSkipImageUpsert(t *testing.T) {
	clairifyScannerID := "id-clairify-scanner"

	createScanner := func(ctrl *gomock.Controller, typ, id string) types.ImageScannerWithDataSource {
		scanner := scannerTypesMocks.NewMockScanner(ctrl)
		scanner.EXPECT().Type().Return(typ)

		imageScannerWithDatasource := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		imageScannerWithDatasource.EXPECT().GetScanner().Return(scanner).AnyTimes()
		imageScannerWithDatasource.EXPECT().DataSource().Return(&storage.DataSource{Id: id}).AnyTimes()
		return imageScannerWithDatasource
	}

	createImageIntegrationSet := func(ctrl *gomock.Controller) integration.Set {
		ssMock := scannerSetMocks.NewMockSet(ctrl)
		ssMock.EXPECT().GetAll().Return([]types.ImageScannerWithDataSource{
			createScanner(ctrl, types.Google, "id-gcr-scanner"),
			createScanner(ctrl, types.ScannerV4, iiStore.DefaultScannerV4Integration.GetId()),
			createScanner(ctrl, types.Clairify, clairifyScannerID),
		})

		iiSet := integrationSetMocks.NewMockSet(ctrl)
		iiSet.EXPECT().ScannerSet().Return(ssMock)

		return iiSet
	}

	createImage := func(imgId, dsId string) *storage.Image {
		return &storage.Image{
			Id:   imgId,
			Name: &storage.ImageName{FullName: fmt.Sprintf("FullName-%v", imgId)},
			Scan: &storage.ImageScan{DataSource: &storage.DataSource{Id: dsId}},
		}
	}

	noDataSourceImage := &storage.Image{}
	scannerV4Image := createImage("id-scannerv4-img", iiStore.DefaultScannerV4Integration.GetId())
	clairifyImage := createImage("id-clairify-img", clairifyScannerID)
	gcrImage := createImage("id-gcr-img", "id-gcr-scanner")

	// These variables exist for readability.
	feature := true
	upsert := true
	wantErr := true
	dbGet := true
	dbImgExists := true
	var noDbImg *storage.Image
	var noDbErr error
	getScanners := true

	testCases := []struct {
		desc             string
		featureEnabled   bool
		upsertExpected   bool
		errExpected      bool
		imageToSave      *storage.Image
		dbGetExpected    bool
		dbImage          *storage.Image
		dbExists         bool
		dbErr            error
		iiSetGetExpected bool
	}{
		{
			"upsert image if cannot determine scanner",
			feature, upsert, !wantErr, noDataSourceImage, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image scanned by scanner v4",
			feature, upsert, !wantErr, scannerV4Image, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image scanned by scanner v4 when image from db scanned by clairify",
			feature, upsert, !wantErr, scannerV4Image, !dbGet, createImage("id", clairifyScannerID), dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image not scanned by clairify nor scanner v4",
			feature, upsert, !wantErr, gcrImage, !dbGet, noDbImg, !dbImgExists, noDbErr, getScanners,
		},
		{
			"upsert image scanned by clairify when db is empty",
			feature, upsert, !wantErr, clairifyImage, dbGet, noDbImg, !dbImgExists, noDbErr, getScanners,
		},
		{
			"upsert image scanned by clairify when image from db not scanned by scanner v4",
			feature, upsert, !wantErr, clairifyImage, dbGet, createImage("id", clairifyScannerID), dbImgExists, noDbErr, getScanners,
		},
		{
			"do not upsert image when scanned by clairify and image from db scanned by scanner v4",
			feature, !upsert, !wantErr, clairifyImage, dbGet, createImage("id", iiStore.DefaultScannerV4Integration.GetId()), dbImgExists, noDbErr, getScanners,
		},
		{
			"do not upsert image when scanned by clairify and getting image from db had error",
			feature, !upsert, wantErr, clairifyImage, dbGet, noDbImg, !dbImgExists, errors.New("fake"), getScanners,
		},
		{
			"upsert image scanned by clairify when scanner v4 feature disabled",
			!feature, upsert, !wantErr, clairifyImage, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.ScannerV4, tc.featureEnabled)
			ctrl := gomock.NewController(t)

			var imageDS *imageDSMocks.MockDataStore
			if tc.dbGetExpected {
				imageDS = imageDSMocks.NewMockDataStore(ctrl)
				imageDS.EXPECT().GetImage(gomock.Any(), tc.imageToSave.GetId()).Return(tc.dbImage, tc.dbExists, tc.dbErr)
			}

			var iiSet integration.Set
			if tc.iiSetGetExpected {
				iiSet = createImageIntegrationSet(ctrl)
			}

			rmService := &managerImpl{
				imageStorage: imageDS,
				iiSet:        iiSet,
			}

			skip, err := rmService.skipImageUpsert(tc.imageToSave)
			if tc.errExpected {
				require.Error(t, err)
				assert.False(t, skip)
			} else {
				require.NoError(t, err)
				assert.Equal(t, !tc.upsertExpected, skip)
			}
		})
	}
}

func TestSkipImageV2Upsert(t *testing.T) {
	pkgTestUtils.MustUpdateFeature(t, features.FlattenImageData, true)
	clairifyScannerID := "id-clairify-scanner"

	createScanner := func(ctrl *gomock.Controller, typ, id string) types.ImageScannerWithDataSource {
		scanner := scannerTypesMocks.NewMockScanner(ctrl)
		scanner.EXPECT().Type().Return(typ)

		imageScannerWithDatasource := scannerTypesMocks.NewMockImageScannerWithDataSource(ctrl)
		imageScannerWithDatasource.EXPECT().GetScanner().Return(scanner).AnyTimes()
		imageScannerWithDatasource.EXPECT().DataSource().Return(&storage.DataSource{Id: id}).AnyTimes()
		return imageScannerWithDatasource
	}

	createImageIntegrationSet := func(ctrl *gomock.Controller) integration.Set {
		ssMock := scannerSetMocks.NewMockSet(ctrl)
		ssMock.EXPECT().GetAll().Return([]types.ImageScannerWithDataSource{
			createScanner(ctrl, types.Google, "id-gcr-scanner"),
			createScanner(ctrl, types.ScannerV4, iiStore.DefaultScannerV4Integration.GetId()),
			createScanner(ctrl, types.Clairify, clairifyScannerID),
		})

		iiSet := integrationSetMocks.NewMockSet(ctrl)
		iiSet.EXPECT().ScannerSet().Return(ssMock)

		return iiSet
	}

	createImage := func(imgId, dsId string) *storage.ImageV2 {
		return &storage.ImageV2{
			Id:   imgId,
			Name: &storage.ImageName{FullName: fmt.Sprintf("FullName-%v", imgId)},
			Scan: &storage.ImageScan{DataSource: &storage.DataSource{Id: dsId}},
		}
	}

	noDataSourceImage := &storage.ImageV2{}
	scannerV4Image := createImage("id-scannerv4-img", iiStore.DefaultScannerV4Integration.GetId())
	clairifyImage := createImage("id-clairify-img", clairifyScannerID)
	gcrImage := createImage("id-gcr-img", "id-gcr-scanner")

	// These variables exist for readability.
	feature := true
	upsert := true
	wantErr := true
	dbGet := true
	dbImgExists := true
	var noDbImg *storage.ImageV2
	var noDbErr error
	getScanners := true

	testCases := []struct {
		desc             string
		featureEnabled   bool
		upsertExpected   bool
		errExpected      bool
		imageToSave      *storage.ImageV2
		dbGetExpected    bool
		dbImage          *storage.ImageV2
		dbExists         bool
		dbErr            error
		iiSetGetExpected bool
	}{
		{
			"upsert image if cannot determine scanner",
			feature, upsert, !wantErr, noDataSourceImage, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image scanned by scanner v4",
			feature, upsert, !wantErr, scannerV4Image, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image scanned by scanner v4 when image from db scanned by clairify",
			feature, upsert, !wantErr, scannerV4Image, !dbGet, createImage("id", clairifyScannerID), dbImgExists, noDbErr, !getScanners,
		},
		{
			"upsert image not scanned by clairify nor scanner v4",
			feature, upsert, !wantErr, gcrImage, !dbGet, noDbImg, !dbImgExists, noDbErr, getScanners,
		},
		{
			"upsert image scanned by clairify when db is empty",
			feature, upsert, !wantErr, clairifyImage, dbGet, noDbImg, !dbImgExists, noDbErr, getScanners,
		},
		{
			"upsert image scanned by clairify when image from db not scanned by scanner v4",
			feature, upsert, !wantErr, clairifyImage, dbGet, createImage("id", clairifyScannerID), dbImgExists, noDbErr, getScanners,
		},
		{
			"do not upsert image when scanned by clairify and image from db scanned by scanner v4",
			feature, !upsert, !wantErr, clairifyImage, dbGet, createImage("id", iiStore.DefaultScannerV4Integration.GetId()), dbImgExists, noDbErr, getScanners,
		},
		{
			"do not upsert image when scanned by clairify and getting image from db had error",
			feature, !upsert, wantErr, clairifyImage, dbGet, noDbImg, !dbImgExists, errors.New("fake"), getScanners,
		},
		{
			"upsert image scanned by clairify when scanner v4 feature disabled",
			!feature, upsert, !wantErr, clairifyImage, !dbGet, noDbImg, !dbImgExists, noDbErr, !getScanners,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.ScannerV4, tc.featureEnabled)
			ctrl := gomock.NewController(t)

			var imageV2DS *imageV2DSMocks.MockDataStore
			if tc.dbGetExpected {
				imageV2DS = imageV2DSMocks.NewMockDataStore(ctrl)
				imageV2DS.EXPECT().GetImage(gomock.Any(), tc.imageToSave.GetId()).Return(tc.dbImage, tc.dbExists, tc.dbErr)
			}

			var iiSet integration.Set
			if tc.iiSetGetExpected {
				iiSet = createImageIntegrationSet(ctrl)
			}

			rmService := &managerImpl{
				imageV2Storage: imageV2DS,
				iiSet:          iiSet,
			}

			skip, err := rmService.skipImageV2Upsert(tc.imageToSave)
			if tc.errExpected {
				require.Error(t, err)
				assert.False(t, skip)
			} else {
				require.NoError(t, err)
				assert.Equal(t, !tc.upsertExpected, skip)
			}
		})
	}
}

func TestReprocessDeploymentRiskUsesCorrectImageID(t *testing.T) {
	testCases := []struct {
		name                    string
		flattenImageDataEnabled bool
		containerImageID        string
		containerImageIDV2      string
		expectedImageIDUsed     string
	}{
		{
			name:                    "uses container image ID when FlattenImageData is disabled",
			flattenImageDataEnabled: false,
			containerImageID:        "sha256:abc123",
			containerImageIDV2:      "uuid-v5-id",
			expectedImageIDUsed:     "sha256:abc123",
		},
		{
			name:                    "uses container image IDV2 when FlattenImageData is enabled",
			flattenImageDataEnabled: true,
			containerImageID:        "sha256:abc123",
			containerImageIDV2:      "uuid-v5-id",
			expectedImageIDUsed:     "uuid-v5-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.FlattenImageData, tc.flattenImageDataEnabled)
			ctrl := gomock.NewController(t)

			deployment := &storage.Deployment{
				Id:   "deployment-id",
				Name: "test-deployment",
				Containers: []*storage.Container{
					{
						Image: &storage.ContainerImage{
							Id:   tc.containerImageID,
							IdV2: tc.containerImageIDV2,
							Name: &storage.ImageName{FullName: "nginx:latest"},
						},
					},
				},
			}

			riskStorageMock := riskDS.NewMockDataStore(ctrl)

			// Expect GetRiskForDeployment to be called
			riskStorageMock.EXPECT().
				GetRiskForDeployment(gomock.Any(), deployment).
				Return(nil, false, nil)

			// Expect GetRisk to be called with the correct image ID based on feature flag
			riskStorageMock.EXPECT().
				GetRisk(gomock.Any(), tc.expectedImageIDUsed, storage.RiskSubjectType_IMAGE).
				Return(&storage.Risk{Score: 5.0}, true, nil)

			// Expect UpsertRisk to be called
			riskStorageMock.EXPECT().
				UpsertRisk(gomock.Any(), gomock.Any()).
				Return(nil)

			deploymentStorageMock := deploymentDS.NewMockDataStore(ctrl)
			deploymentStorageMock.EXPECT().
				UpsertDeployment(gomock.Any(), gomock.Any()).
				Return(nil)

			// Create mock alert searcher that returns nil results
			mockAlertSearcher := &getters.MockAlertsSearcher{
				Alerts: nil,
			}

			// Create mock evaluator that returns nil results
			mockEvaluator := evaluatorMocks.NewMockEvaluator(ctrl)
			mockEvaluator.EXPECT().
				EvaluateBaselinesAndPersistResult(gomock.Any()).
				Return(nil, nil).
				AnyTimes()

			// Create the actual deployment scorer with mocked dependencies
			scorer := deploymentScorer.NewDeploymentScorer(mockAlertSearcher, mockEvaluator)

			manager := &managerImpl{
				riskStorage:       riskStorageMock,
				deploymentScorer:  scorer,
				deploymentStorage: deploymentStorageMock,
				clusterRanker:     ranking.NewRanker(),
				nsRanker:          ranking.NewRanker(),
			}

			manager.ReprocessDeploymentRisk(deployment)
		})
	}
}
