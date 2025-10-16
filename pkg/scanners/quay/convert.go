package quay

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/stringutils"
)

func convertScanToImageScan(image *storage.Image, s *scanResult) *storage.ImageScan {
	os := stringutils.OrDefault(s.Data.Layer.NamespaceName, "unknown")
	imageScan := &storage.ImageScan{}
	imageScan.SetOperatingSystem(os)
	imageScan.SetScanTime(protocompat.TimestampNow())
	imageScan.SetComponents(clair.ConvertFeatures(image, s.Data.Layer.Features, os))
	return imageScan
}
