package quay

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/clair"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

func convertScanToImageScan(image *storage.Image, s *scanResult) *storage.ImageScan {
	return &storage.ImageScan{
		OperatingSystem: stringutils.OrDefault(s.Data.Layer.NamespaceName, "unknown"),
		ScanTime:        types.TimestampNow(),
		Components:      clair.ConvertFeatures(image, s.Data.Layer.Features),
	}
}
