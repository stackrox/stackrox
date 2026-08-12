package clairify

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair/mock"
	"github.com/stackrox/rox/pkg/protoassert"
	clairV1 "github.com/stackrox/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func getTestScan() (*clairV1.LayerEnvelope, *storage.ImageScan, *storage.Image) {
	scannerVersion := "2.22.0"

	image := &storage.Image{
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "integration/nginx",
			Tag:      "1.10",
		},
	}
	clairFeatures, protoComponents := mock.GetTestFeatures()

	env := clairV1.LayerEnvelope{
		Layer: &clairV1.Layer{
			NamespaceName: "debian:8",
			Features:      clairFeatures,
		},
		ScannerVersion: scannerVersion,
	}

	protoScan := &storage.ImageScan{
		Components:      protoComponents,
		ScannerVersion:  scannerVersion,
		OperatingSystem: "debian:8",
		Notes: []storage.ImageScan_Note{
			storage.ImageScan_OS_CVES_STALE,
		},
	}
	return &env, protoScan, image
}

func TestClairifyClose_NilConnection(t *testing.T) {
	scanner := &clairify{}
	assert.NoError(t, scanner.Close())
}

func TestClairifyClose_ClosesGRPCConnection(t *testing.T) {
	conn, err := grpc.Dial("localhost:0", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	scanner := &clairify{gRPCConnection: conn}
	require.NoError(t, scanner.Close())

	assert.Equal(t, connectivity.Shutdown, conn.GetState())
}

func TestConvertLayerToImageScan(t *testing.T) {
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
