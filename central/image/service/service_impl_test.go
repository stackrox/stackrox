package service

import (
	"testing"

	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestBuildNames(t *testing.T) {
	srcImage := &storage.ImageName{FullName: "si"}

	t.Run("nil metadata", func(t *testing.T) {
		names := buildNames(srcImage, nil, nil)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("empty metadata", func(t *testing.T) {
		names := buildNames(srcImage, nil, &storage.ImageMetadata{})
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with empty data source", func(t *testing.T) {
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{}}
		names := buildNames(srcImage, nil, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image:latest"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, nil, metadata)
		assert.Len(t, names, 2)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
		assert.Equal(t, mirror, names[1].GetFullName())
	})

	t.Run("metadata with invalid mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image@sha256:bad"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, nil, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("existing names and mirror", func(t *testing.T) {
		existingNames := []*storage.ImageName{
			{FullName: "si"}, // Dupe should be omitted
			{FullName: "e1"},
			{FullName: "e2"},
			{FullName: "si"}, // Dupe should be omitted
		}
		mirror := "example.com/mirror/image:latest"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}

		names := buildNames(srcImage, existingNames, metadata)
		require.Len(t, names, 4)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
		assert.Equal(t, existingNames[1].GetFullName(), names[1].GetFullName())
		assert.Equal(t, existingNames[2].GetFullName(), names[2].GetFullName())
		assert.Equal(t, mirror, names[3].GetFullName())
	})
}

func TestShouldUpdateExistingScan(t *testing.T) {
	// These variables exist for readability.
	var noExistingImg *storage.Image
	var emptyReq *v1.EnrichLocalImageInternalRequest
	feature := true
	update := true
	imgExists := true
	v4DataSource := &storage.DataSource{Id: iiStore.DefaultScannerV4Integration.GetId()}
	v4MatchReq := &v1.EnrichLocalImageInternalRequest{IndexerVersion: "v4"}
	v2MatchReq := &v1.EnrichLocalImageInternalRequest{}
	v2ExpiredScan := &storage.Image{Scan: &storage.ImageScan{ScanTime: protoconv.NowMinus(reprocessInterval * 2)}}
	v2CurrentScan := &storage.Image{Scan: &storage.ImageScan{ScanTime: protoconv.NowMinus(0)}}
	v4ExpiredScan := &storage.Image{Scan: &storage.ImageScan{ScanTime: v2ExpiredScan.Scan.ScanTime, DataSource: v4DataSource}}
	v4CurrentScan := &storage.Image{Scan: &storage.ImageScan{ScanTime: protoconv.NowMinus(0), DataSource: v4DataSource}}

	testCases := []struct {
		desc           string
		featureEnabled bool
		imgExists      bool
		existingImg    *storage.Image
		req            *v1.EnrichLocalImageInternalRequest
		expected       bool
	}{
		// Scanner V4 feature disabled.
		{
			"update if no existing image",
			!feature, !imgExists, noExistingImg, emptyReq, update,
		},
		{
			"update if no existing scan",
			!feature, imgExists, &storage.Image{}, emptyReq, update,
		},
		{
			"update if scan expired",
			!feature, imgExists, v2ExpiredScan, emptyReq, update,
		},
		{
			"no update if scan is current",
			!feature, imgExists, v2CurrentScan, emptyReq, !update,
		},
		// Scanner V4 feature enabled.
		{
			"update if no existing image (feature enabled)",
			feature, !imgExists, noExistingImg, emptyReq, update,
		},
		{
			"update if no existing scan (feature enabled)",
			feature, imgExists, &storage.Image{}, emptyReq, update,
		},
		{
			"update if v2 scan expired and match request for v4",
			feature, imgExists, v2ExpiredScan, v4MatchReq, update,
		},
		{
			"update if v2 scan expired and match request for v2",
			feature, imgExists, v2ExpiredScan, v2MatchReq, update,
		},
		{
			"no update if v2 scan NOT expired and match request for v2",
			feature, imgExists, v2CurrentScan, v2MatchReq, !update,
		},
		{
			"update if v4 scan expired and match request for v4",
			feature, imgExists, v4ExpiredScan, v4MatchReq, update,
		},
		{
			"no update if v4 scan NOT expired and match request for v4",
			feature, imgExists, v4CurrentScan, v4MatchReq, !update,
		},
		{
			"no update if v4 scan expired and match request for v2",
			feature, imgExists, v4ExpiredScan, v2MatchReq, !update,
		},
		{
			"update if v2 scan NOT expired and match request for v4",
			feature, imgExists, v2CurrentScan, v4MatchReq, update,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.ScannerV4, tc.featureEnabled)

			actual := shouldUpdateExistingScan(tc.imgExists, tc.existingImg, tc.req)
			assert.Equal(t, tc.expected, actual)
		})
	}

}

func TestUpdatingImageFromRequest(t *testing.T) {
	createImgName := func(name string) *storage.ImageName {
		imgName, _, err := utils.GenerateImageNameFromString(name)
		if err != nil {
			t.Fatal(err)
		}
		return imgName
	}

	imgAName := createImgName("docker.io/library/nginx:latest")
	imgBName := createImgName("example.com/library/nginx:latest")   // diff registry
	imgCName := createImgName("docker.io/different/nginx:latest")   // diff remote
	imgDName := createImgName("example.com/different/nginx:latest") // diff registry and remote

	imgA := &storage.Image{Name: imgAName}
	imgAWithMeta := &storage.Image{Name: imgAName, Metadata: &storage.ImageMetadata{}}

	tcs := []struct {
		name         string
		existingImg  *storage.Image
		reqImgName   *storage.ImageName
		expectedName *storage.ImageName
		feature      bool
	}{
		{
			"feature disabled do not update name",
			imgA, imgBName, imgAName, false,
		},
		{
			"metadata exists do not update name",
			imgAWithMeta, imgBName, imgAName, true,
		},
		{
			"images are the same do not update name",
			imgA, imgAName, imgAName, true,
		},
		{
			"registry differs update name",
			imgA, imgBName, imgBName, true,
		},
		{
			"remote differs update name",
			imgA, imgCName, imgCName, true,
		},
		{
			"registry and remote differs update name",
			imgA, imgDName, imgDName, true,
		},
		{
			"image name nil do not update name",
			imgA, nil, imgAName, true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			pkgTestUtils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, tc.feature)

			clone := tc.existingImg.CloneVT()
			updateImageFromRequest(clone, tc.reqImgName)
			protoassert.Equal(t, tc.expectedName, clone.Name)
		})
	}
}
