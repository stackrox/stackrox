package utils

import (
	"github.com/stackrox/rox/generated/storage"
)

func ConvertToV1(image *storage.ImageV2) *storage.Image {
	if image == nil {
		return nil
	}
	image2 := &storage.Image{}
	image2.SetId(image.GetDigest())
	image2.SetName(image.GetName())
	image2.SetNames([]*storage.ImageName{image.GetName()})
	image2.SetIsClusterLocal(image.GetIsClusterLocal())
	image2.SetLastUpdated(image.GetLastUpdated())
	image2.SetMetadata(image.GetMetadata())
	image2.SetNotes(ConvertNotesToV1(image.GetNotes()))
	image2.SetNotPullable(image.GetNotPullable())
	image2.SetPriority(image.GetPriority())
	image2.SetRiskScore(image.GetRiskScore())
	image2.SetScan(image.GetScan())
	image2.Set_Components(image.GetScanStats().GetComponentCount())
	image2.Set_Cves(image.GetScanStats().GetCveCount())
	image2.SetFixableCves(image.GetScanStats().GetFixableCveCount())
	image2.Set_TopCvss(image.GetTopCvss())
	image2.SetSignature(image.GetSignature())
	image2.SetSignatureVerificationData(image.GetSignatureVerificationData())
	return image2
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
	ret := &storage.ImageV2{}
	ret.SetId(NewImageV2ID(image.GetName(), image.GetId()))
	ret.SetDigest(image.GetId())
	ret.SetName(image.GetName())
	ret.SetIsClusterLocal(image.GetIsClusterLocal())
	ret.SetLastUpdated(image.GetLastUpdated())
	ret.SetMetadata(image.GetMetadata())
	ret.SetNotes(ConvertNotesToV2(image.GetNotes()))
	ret.SetNotPullable(image.GetNotPullable())
	ret.SetPriority(image.GetPriority())
	ret.SetRiskScore(image.GetRiskScore())
	ret.SetScan(image.GetScan())
	ret.SetTopCvss(image.GetTopCvss())
	ret.SetSignatureVerificationData(image.GetSignatureVerificationData())
	ret.SetSignature(image.GetSignature())
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
