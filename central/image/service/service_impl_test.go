package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
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

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestBuildNames(t *testing.T) {
	srcImage := &storage.ImageName{FullName: "si"}

	t.Run("nil metadata", func(t *testing.T) {
		names := buildNames(srcImage, nil)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("empty metadata", func(t *testing.T) {
		names := buildNames(srcImage, &storage.ImageMetadata{})
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with empty data source", func(t *testing.T) {
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image:latest"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 2)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
		assert.Equal(t, mirror, names[1].GetFullName())
	})

	t.Run("metadata with invalid mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image@sha256:bad"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})
}

func TestSaveImage(t *testing.T) {
	ctx := context.Background()

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
			createScanner(ctrl, "google", "id-gcr-scanner"),
			createScanner(ctrl, "scannerv4", iiStore.DefaultScannerV4Integration.GetId()),
			createScanner(ctrl, "clairify", clairifyScannerID),
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
			"upsert image not scanned by clairify or scanner v4",
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
			pkgTestUtils.MustUpdateFeature(t, features.ScannerV4Enabled, tc.featureEnabled)
			ctrl := gomock.NewController(t)

			var rm *riskManagerMocks.MockManager
			if tc.upsertExpected {
				rm = riskManagerMocks.NewMockManager(ctrl)
				rm.EXPECT().CalculateRiskAndUpsertImage(tc.imageToSave).Return(nil)
			}

			var imageDS *imageDSMocks.MockDataStore
			if tc.dbGetExpected {
				imageDS = imageDSMocks.NewMockDataStore(ctrl)
				imageDS.EXPECT().GetImage(ctx, tc.imageToSave.GetId()).Return(tc.dbImage, tc.dbExists, tc.dbErr)
			}

			var iiSet integration.Set
			if tc.iiSetGetExpected {
				iiSet = createImageIntegrationSet(ctrl)
			}

			imageService := &serviceImpl{
				riskManager:    rm,
				datastore:      imageDS,
				integrationSet: iiSet,
			}

			err := imageService.saveImage(ctx, tc.imageToSave)
			if tc.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})

	}
}
