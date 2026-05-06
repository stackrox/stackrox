package utils

import (
	"testing"

	deploymentDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/views"
	imageV2DSMocks "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// expectCapableConn sets up a mock connection that has the capability and
// expects both UpdatedImage and AC-only InvalidateImageCache messages.
func expectCapableConn(ctrl *gomock.Controller, connMgr *connMocks.MockManager, clusterID string) {
	conn := connMocks.NewMockSensorConnection(ctrl)
	connMgr.EXPECT().GetConnection(clusterID).Return(conn)
	conn.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).Return(true)
	conn.EXPECT().InjectMessage(gomock.Any(), gomock.Cond(func(m *central.MsgToSensor) bool {
		return m.GetUpdatedImage() != nil
	})).Return(nil)
	conn.EXPECT().InjectMessage(gomock.Any(), gomock.Cond(func(m *central.MsgToSensor) bool {
		inv := m.GetInvalidateImageCache()
		return inv != nil && inv.GetAdmissionControllerOnly()
	})).Return(nil)
}

// expectCapableConnCapturingSentImage is like expectCapableConn but captures
// the image sent via UpdatedImage for assertion.
func expectCapableConnCapturingSentImage(ctrl *gomock.Controller, connMgr *connMocks.MockManager, clusterID string, sentImage **storage.Image) {
	conn := connMocks.NewMockSensorConnection(ctrl)
	connMgr.EXPECT().GetConnection(clusterID).Return(conn)
	conn.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).Return(true)
	conn.EXPECT().InjectMessage(gomock.Any(), gomock.Cond(func(m *central.MsgToSensor) bool {
		if m.GetUpdatedImage() != nil {
			*sentImage = m.GetUpdatedImage()
			return true
		}
		return false
	})).Return(nil)
	conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Return(nil)
}

func testImageName() *storage.ImageName {
	return &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "1.25",
		FullName: "docker.io/library/nginx:1.25",
	}
}

func TestUpdateImageCaches_NilKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	connMgr := connMocks.NewMockManager(ctrl)
	UpdateImageCaches(connMgr, nil, nil, &storage.Image{}, nil)
}

func TestUpdateImageCaches_ByName(t *testing.T) {
	for name, tc := range map[string]struct {
		clusterCount  int
		hasCapability bool
	}{
		"sends to capable cluster":   {clusterCount: 1, hasCapability: true},
		"skips incapable cluster":    {clusterCount: 1, hasCapability: false},
		"sends to multiple clusters": {clusterCount: 2, hasCapability: true},
	} {
		t.Run(name, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.FlattenImageData, false)
			ctrl := gomock.NewController(t)
			connMgr := connMocks.NewMockManager(ctrl)
			deploymentDS := deploymentDSMocks.NewMockDataStore(ctrl)

			clusterIDs := make([]string, tc.clusterCount)
			for i := range tc.clusterCount {
				clusterIDs[i] = "cluster-" + string(rune('A'+i))
				if tc.hasCapability {
					expectCapableConn(ctrl, connMgr, clusterIDs[i])
				} else {
					conn := connMocks.NewMockSensorConnection(ctrl)
					connMgr.EXPECT().GetConnection(clusterIDs[i]).Return(conn)
					conn.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).Return(false)
				}
			}

			imageSHA := "sha256:abc"
			deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return(
				[]*views.ContainerImageView{{ImageNameFullName: "docker.io/library/nginx:1.25", ClusterIDs: clusterIDs}}, nil,
			)

			img := &storage.Image{Id: imageSHA, Name: testImageName()}
			UpdateImageCaches(connMgr, deploymentDS, nil, img,
				&central.ImageKey{ImageId: imageSHA, ImageFullName: "docker.io/library/nginx:1.25"})
		})
	}
}

func TestUpdateImageCaches_DigestOnlyFallback(t *testing.T) {
	for name, tc := range map[string]struct {
		flattenEnabled bool
		searchField    string
	}{
		"V1 falls back to ImageSHA": {flattenEnabled: false},
		"V2 falls back to ImageID":  {flattenEnabled: true},
	} {
		t.Run(name, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.FlattenImageData, tc.flattenEnabled)
			ctrl := gomock.NewController(t)
			connMgr := connMocks.NewMockManager(ctrl)
			deploymentDS := deploymentDSMocks.NewMockDataStore(ctrl)

			clusterID := "cluster-A"
			imageSHA := "sha256:abc"
			imageIDV2 := "v2-uuid-test"

			deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return(
				[]*views.ContainerImageView{{ClusterIDs: []string{clusterID}}}, nil,
			)

			if tc.flattenEnabled {
				connMgr.EXPECT().AllSensorsHaveCapability(gomock.Any()).Return(true)
			}
			expectCapableConn(ctrl, connMgr, clusterID)

			img := &storage.Image{
				Id: imageSHA,
				Name: &storage.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					FullName: "docker.io/library/nginx@sha256:abc",
				},
			}
			UpdateImageCaches(connMgr, deploymentDS, nil, img,
				&central.ImageKey{ImageId: imageSHA, ImageIdV2: imageIDV2, ImageFullName: "docker.io/library/nginx@sha256:abc"})
		})
	}
}

func TestUpdateImageCaches_V2BackwardCompat(t *testing.T) {
	for name, tc := range map[string]struct {
		allSensorsHaveFlatten bool
		extraNames            []*storage.ImageName
		wantMinNames          int
	}{
		"all sensors have FlattenImageData": {allSensorsHaveFlatten: true, wantMinNames: 1},
		"backward compat fetches all names": {
			allSensorsHaveFlatten: false,
			extraNames:            []*storage.ImageName{{FullName: "nginx:latest"}, {FullName: "nginx:1.25"}},
			wantMinNames:          2,
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.FlattenImageData, true)
			ctrl := gomock.NewController(t)
			connMgr := connMocks.NewMockManager(ctrl)
			deploymentDS := deploymentDSMocks.NewMockDataStore(ctrl)
			imageV2DS := imageV2DSMocks.NewMockDataStore(ctrl)

			imageSHA := "sha256:v2test"
			imageIDV2 := "v2-uuid-test"
			clusterID := "cluster-v2"

			deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return(
				[]*views.ContainerImageView{{ImageIDV2: imageIDV2, ImageDigest: imageSHA, ClusterIDs: []string{clusterID}}}, nil,
			)
			connMgr.EXPECT().AllSensorsHaveCapability(gomock.Any()).Return(tc.allSensorsHaveFlatten)
			if !tc.allSensorsHaveFlatten {
				imageV2DS.EXPECT().GetImageNames(gomock.Any(), imageSHA).Return(tc.extraNames, nil)
			}

			var sentImage *storage.Image
			expectCapableConnCapturingSentImage(ctrl, connMgr, clusterID, &sentImage)

			img := &storage.Image{
				Id:    imageSHA,
				Name:  testImageName(),
				Names: []*storage.ImageName{{FullName: "nginx:latest"}},
			}
			UpdateImageCaches(connMgr, deploymentDS, imageV2DS, img,
				&central.ImageKey{ImageId: imageSHA, ImageIdV2: imageIDV2, ImageFullName: "docker.io/library/nginx:1.25"})

			assert.GreaterOrEqual(t, len(sentImage.GetNames()), tc.wantMinNames)
		})
	}
}
