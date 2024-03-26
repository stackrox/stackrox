package service

import (
	"testing"

	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoconv"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
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
