package types

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
)

// GenericImage is an interface that implements the common functions of Image and ContainerImage
type GenericImage interface {
	GetId() string
	GetName() *storage.ImageName
}

// ToImage converts a storage.ContainerImage to a storage.Image
func ToImage(ci *storage.ContainerImage) *storage.Image {
	image := &storage.Image{}
	image.SetId(ci.GetId())
	image.SetName(ci.GetName())
	image.SetNames([]*storage.ImageName{ci.GetName()})
	image.SetNotPullable(ci.GetNotPullable())
	image.SetIsClusterLocal(ci.GetIsClusterLocal())
	return image
}

// ToImageV2 converts a storage.ContainerImage to a storage.ImageV2
func ToImageV2(ci *storage.ContainerImage) *storage.ImageV2 {
	imageV2 := &storage.ImageV2{}
	imageV2.SetId(ci.GetIdV2())
	imageV2.SetName(ci.GetName())
	imageV2.SetNotPullable(ci.GetNotPullable())
	imageV2.SetIsClusterLocal(ci.GetIsClusterLocal())
	return imageV2
}

// ToContainerImage converts a storage.Image to a storage.ContainerImage
func ToContainerImage(ci *storage.Image) *storage.ContainerImage {
	res := &storage.ContainerImage{}
	res.SetId(ci.GetId())
	res.SetName(ci.GetName())
	res.SetNotPullable(ci.GetNotPullable())
	if features.FlattenImageData.Enabled() && ci.GetId() != "" {
		res.SetIdV2(utils.NewImageV2ID(ci.GetName(), ci.GetId()))
	}
	return res
}

// ToContainerImageV2 converts a storage.ImageV2 to a storage.ContainerImage
func ToContainerImageV2(ci *storage.ImageV2) *storage.ContainerImage {
	ci2 := &storage.ContainerImage{}
	ci2.SetIdV2(ci.GetId())
	ci2.SetName(ci.GetName())
	ci2.SetNotPullable(ci.GetNotPullable())
	return ci2
}

// ConvertImageToListImage converts an image to a ListImage
func ConvertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{}
	listImage.SetId(i.GetId())
	listImage.SetName(i.GetName().GetFullName())
	listImage.SetCreated(i.GetMetadata().GetV1().GetCreated())
	listImage.SetLastUpdated(i.GetLastUpdated())
	if i.GetSetComponents() != nil {
		listImage.Set_Components(i.GetComponents())
	}
	if i.GetSetCves() != nil {
		listImage.Set_Cves(i.GetCves())
	}
	if i.GetSetFixable() != nil {
		listImage.SetFixableCves(i.GetFixableCves())
	}
	return listImage
}
