package utils

import (
	"github.com/stackrox/rox/generated/storage"
)

func ConvertToV1(image *storage.ImageV2) *storage.Image {
	if image == nil {
		return nil
	}
	return &storage.Image{
		Id:             image.GetDigest(),
		Name:           image.GetName(),
		Names:          []*storage.ImageName{image.GetName()},
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
	}
}

func ConvertNotesToV1(notes []storage.ImageV2_Note) []storage.Image_Note {
	res := make([]storage.Image_Note, 0)
	for _, note := range notes {
		res = append(res, storage.Image_Note(note.Number()))
	}
	return res
}

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
