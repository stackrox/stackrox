package mapper

import "github.com/stackrox/rox/generated/storage"

func ConvertToV1(i *storage.ImageV2) *storage.Image {
	return &storage.Image{
		Id:             i.Sha,
		Name:           i.Name,
		Names:          []*storage.ImageName{i.Name},
		IsClusterLocal: i.IsClusterLocal,
		LastUpdated:    i.LastUpdated,
		Metadata:       i.Metadata,
		Notes:          ConvertNotesToV1(i.Notes),
		NotPullable:    i.NotPullable,
		Priority:       i.Priority,
		RiskScore:      i.RiskScore,
		Scan:           i.Scan,
		SetComponents: &storage.Image_Components{
			Components: i.ComponentCount,
		},
		SetCves: &storage.Image_Cves{
			Cves: i.CveCount,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: i.FixableCveCount,
		},
		SetTopCvss: &storage.Image_TopCvss{
			TopCvss: i.TopCvss,
		},
		Signature:                 i.Signature,
		SignatureVerificationData: i.SignatureVerificationData,
	}
}

func ConvertNotesToV1(i []storage.ImageV2_Note) []storage.Image_Note {
	notes := make([]storage.Image_Note, 0)
	for _, note := range i {
		notes = append(notes, storage.Image_Note(note.Number()))
	}
	return notes
}
