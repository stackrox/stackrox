package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/assert"
)

func getTestScan() (*clairV1.LayerEnvelope, *storage.ImageScan, *storage.Image) {
	scannerVersion := "2.22.0"

	imageName := &storage.ImageName{}
	imageName.SetRegistry("docker.io")
	imageName.SetRemote("integration/nginx")
	imageName.SetTag("1.10")
	image := &storage.Image{}
	image.SetName(imageName)
	clairFeatures, protoComponents := mock.GetTestFeatures()

	env := clairV1.LayerEnvelope{
		Layer: &clairV1.Layer{
			NamespaceName: "debian:8",
			Features:      clairFeatures,
		},
		ScannerVersion: scannerVersion,
	}

	protoScan := &storage.ImageScan{}
	protoScan.SetComponents(protoComponents)
	protoScan.SetScannerVersion(scannerVersion)
	protoScan.SetOperatingSystem("debian:8")
	protoScan.SetNotes([]storage.ImageScan_Note{
		storage.ImageScan_OS_CVES_STALE,
	})
	return &env, protoScan, image
}

func TestConvertLayerToImageScan(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	layer, protoScan, image := getTestScan()
	actualScan := convertLayerToImageScan(image, layer)
	// Ignore Scan time in the test, as it is defined as the time we retrieve the scan.
	protoassert.Equal(t, protoScan.GetDataSource(), actualScan.GetDataSource())
	assert.Equal(t, "debian:8", actualScan.GetOperatingSystem())
	protoassert.SlicesEqual(t, protoScan.GetComponents(), actualScan.GetComponents())
	assert.Equal(t, protoScan.GetScannerVersion(), actualScan.GetScannerVersion())
	assert.Len(t, protoScan.GetNotes(), 1)
	assert.Contains(t, protoScan.GetNotes(), convertNote(clairV1.OSCVEsStale))
}
