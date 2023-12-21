package quay

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/stringutils"
)

func convertScanToImageScan(image *storage.Image, s *scanResult) *storage.ImageScan {
	os := stringutils.OrDefault(s.Data.Layer.NamespaceName, "unknown")
	return &storage.ImageScan{
		OperatingSystem: os,
		ScanTime:        protoconv.TimestampNow(),
		Components:      clair.ConvertFeatures(image, s.Data.Layer.Features, os),
	}
}
