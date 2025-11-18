package service

import (
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/timestamp"
)

func internalScanRespFromImage(img *storage.Image) *v1.ScanImageInternalResponse {
	utils.FilterSuppressedCVEsNoClone(img)
	utils.StripCVEDescriptionsNoClone(img)
	return &v1.ScanImageInternalResponse{
		Image: img,
	}
}

// scanExpired returns true when the scan associated with the image
// is considered expired.
func scanExpired(imgScan *storage.ImageScan) bool {
	scanTime := timestamp.FromProtobuf(imgScan.GetScanTime())
	return !scanTime.Add(reprocessInterval).After(timestamp.Now())
}

// updateImageFromRequest will update the name of existing image with the one from the request
// if the names differ and the metadata for the existing image was unable to be pulled previously.
// Returns true if an update was made, false otherwise.
func updateImageFromRequest(existingImg *storage.Image, reqImgName *storage.ImageName) bool {
	if !features.UnqualifiedSearchRegistries.Enabled() || reqImgName == nil {
		// The need for this behavior is associated with the use of unqualified search
		// registries or short name aliases (currently), if the feature is disabled
		// do not modify the name.
		return false
	}

	if existingImg.GetMetadata() != nil {
		// If metadata exists, then the existing image name is likely valid, no update needed.
		return false
	}

	existingImgName := existingImg.GetName()
	if existingImgName.GetRegistry() == reqImgName.GetRegistry() &&
		existingImgName.GetRemote() == reqImgName.GetRemote() {
		// No updated needed.
		return false
	}

	// If the existing image had missing metadata and this request has a different registry or
	// remote it's possible the values were incorrect when Sensor sent the initial request to Central.
	// This could occur when unqualified search registries or short name aliases are in use due
	// to the actual registry/repo/digest not being known until the container runtime pulls the image.

	// Replace the image name with the one from the request since it is more likely to be 'correct'.
	log.Debugf("Updated existing image name from %q to %q", existingImgName.GetFullName(), reqImgName.GetFullName())
	existingImg.Name = reqImgName

	return true
}

// shouldUpdateExistingScan will return true if an image should be scanned / re-scanned, false otherwise.
func shouldUpdateExistingScan(imgExists bool, existingScan *storage.ImageScan, request *v1.EnrichLocalImageInternalRequest) bool {
	if !imgExists || existingScan == nil {
		return true
	}

	if !features.ScannerV4.Enabled() {
		return scanExpired(existingScan)
	}

	v4MatchRequest := scannerTypes.ScannerV4IndexerVersion(request.GetIndexerVersion())
	v4ExistingScan := existingScan.GetDataSource().GetId() == iiStore.DefaultScannerV4Integration.GetId()
	if v4ExistingScan && !v4MatchRequest {
		// Do not overwrite a V4 scan with a Clairify scan regardless of expiration.
		log.Debugf("Not updating cached Scanner V4 scan with Clairify scan for image %q", request.GetImageName().GetFullName())
		return false
	}

	if !v4ExistingScan && v4MatchRequest {
		// If the existing scan is NOT from Scanner V4 but the request is from Scanner V4,
		// then scan regardless of expiration.
		log.Debugf("Forcing overwrite of cached Clairify scan with Scanner V4 scan for image %q", request.GetImageName().GetFullName())
		return true
	}

	return scanExpired(existingScan)
}

// buildNames returns a slice containing the known image names from the various parameters.
func buildNames(requestImageName *storage.ImageName, existingImageNames []*storage.ImageName, metadata *storage.ImageMetadata) []*storage.ImageName {
	names := []*storage.ImageName{requestImageName}
	names = append(names, existingImageNames...)

	// Add a mirror name if exists.
	if mirror := metadata.GetDataSource().GetMirror(); mirror != "" {
		mirrorImg, err := utils.GenerateImageFromString(mirror)
		if err != nil {
			log.Warnw("Failed generating image from string",
				logging.String("mirror", mirror), logging.Err(err))
		} else {
			names = append(names, mirrorImg.GetName())
		}
	}

	names = protoutils.SliceUnique(names)
	return names
}
