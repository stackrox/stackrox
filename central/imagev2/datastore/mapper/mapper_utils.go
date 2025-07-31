package mapper

import "github.com/stackrox/rox/generated/storage"

func ConvertToV1(image *storage.ImageV2) *storage.Image {
	return &storage.Image{
		Id:             image.Sha,
		Name:           image.Name,
		Names:          []*storage.ImageName{image.Name},
		IsClusterLocal: image.IsClusterLocal,
		LastUpdated:    image.LastUpdated,
		Metadata:       image.Metadata,
		Notes:          ConvertNotesToV1(image.Notes),
		NotPullable:    image.NotPullable,
		Priority:       image.Priority,
		RiskScore:      image.RiskScore,
		Scan:           image.Scan,
		SetComponents: &storage.Image_Components{
			Components: image.ComponentCount,
		},
		SetCves: &storage.Image_Cves{
			Cves: image.CveCount,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: image.FixableCveCount,
		},
		SetTopCvss: &storage.Image_TopCvss{
			TopCvss: image.TopCvss,
		},
		Signature:                 image.Signature,
		SignatureVerificationData: image.SignatureVerificationData,
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
	return &storage.ImageV2{
		Id:                        image.Id,
		Sha:                       image.Id,
		Name:                      image.Name,
		IsClusterLocal:            image.IsClusterLocal,
		LastUpdated:               image.LastUpdated,
		Metadata:                  image.Metadata,
		Notes:                     ConvertNotesToV2(image.Notes),
		NotPullable:               image.NotPullable,
		Priority:                  image.Priority,
		RiskScore:                 image.RiskScore,
		Scan:                      image.Scan,
		ComponentCount:            image.GetComponents(),
		CveCount:                  image.GetCves(),
		FixableCveCount:           image.GetFixableCves(),
		TopCvss:                   image.GetTopCvss(),
		SignatureVerificationData: image.SignatureVerificationData,
		Signature:                 image.Signature,
	}
}

func ConvertNotesToV2(notes []storage.Image_Note) []storage.ImageV2_Note {
	res := make([]storage.ImageV2_Note, 0)
	for _, note := range notes {
		res = append(res, storage.ImageV2_Note(note.Number()))
	}
	return res
}
