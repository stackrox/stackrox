package utils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// ConvertToV1 converts a storage.ImageV2 to a storage.Image.
// If names are provided, they will be added to the converted imageV1's names along with the given imageV2's name.
func ConvertToV1(image *storage.ImageV2, names ...*storage.ImageName) *storage.Image {
	if image == nil {
		return nil
	}
	// Use provided names if available, otherwise default to just the image's name
	imageNames := []*storage.ImageName{image.GetName()}
	if len(names) > 0 {
		imageNames = sliceutils.Unique(append(imageNames, names...))
	}
	return &storage.Image{
		Id:             image.GetDigest(),
		Name:           image.GetName(),
		Names:          imageNames,
		IsClusterLocal: image.GetIsClusterLocal(),
		LastUpdated:    image.GetLastUpdated(),
		Metadata:       image.GetMetadata(),
		Notes:          ConvertNotesToV1(image.GetNotes()),
		NotPullable:    image.GetNotPullable(),
		Priority:       image.GetPriority(),
		RiskScore:      image.GetRiskScore(),
		Scan:           image.GetScan(),
		SetComponents: &storage.Image_Components{
			Components: image.GetScanStats().GetComponentCount(),
		},
		SetCves: &storage.Image_Cves{
			Cves: image.GetScanStats().GetCveCount(),
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: image.GetScanStats().GetFixableCveCount(),
		},
		SetTopCvss: &storage.Image_TopCvss{
			TopCvss: image.GetTopCvss(),
		},
		Signature:                 image.GetSignature(),
		SignatureVerificationData: image.GetSignatureVerificationData(),
		BaseImageInfo:             image.GetBaseImageInfo(),
	}
}

// ConvertNotesToV1 converts a list of storage.ImageV2_Note to a list of storage.Image_Note.
func ConvertNotesToV1(notes []storage.ImageV2_Note) []storage.Image_Note {
	res := make([]storage.Image_Note, 0)
	for _, note := range notes {
		res = append(res, storage.Image_Note(note.Number()))
	}
	return res
}

// ConvertToV1List converts a list of storage.ImageV2 to a list of storage.Image.
// Note that this function does not populate all known names for each imageV2 SHA.
func ConvertToV1List(imagesV2 []*storage.ImageV2) []*storage.Image {
	res := make([]*storage.Image, 0, len(imagesV2))
	for _, imageV2 := range imagesV2 {
		if imageV2 == nil {
			continue
		}
		res = append(res, ConvertToV1(imageV2))
	}
	return res
}

// ConvertToV2 converts a storage.Image to a storage.ImageV2.
func ConvertToV2(image *storage.Image) *storage.ImageV2 {
	if image == nil {
		return nil
	}
	ret := &storage.ImageV2{
		Id:                        NewImageV2ID(image.GetName(), image.GetId()),
		Digest:                    image.GetId(),
		Name:                      image.GetName(),
		IsClusterLocal:            image.GetIsClusterLocal(),
		LastUpdated:               image.GetLastUpdated(),
		Metadata:                  image.GetMetadata(),
		Notes:                     ConvertNotesToV2(image.GetNotes()),
		NotPullable:               image.GetNotPullable(),
		Priority:                  image.GetPriority(),
		RiskScore:                 image.GetRiskScore(),
		Scan:                      image.GetScan(),
		TopCvss:                   image.GetTopCvss(),
		SignatureVerificationData: image.GetSignatureVerificationData(),
		Signature:                 image.GetSignature(),
		BaseImageInfo:             image.GetBaseImageInfo(),
	}
	FillScanStatsV2(ret)
	return ret
}

func ConvertNotesToV2(notes []storage.Image_Note) []storage.ImageV2_Note {
	res := make([]storage.ImageV2_Note, 0)
	for _, note := range notes {
		res = append(res, storage.ImageV2_Note(note.Number()))
	}
	return res
}
