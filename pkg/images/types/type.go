package types

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
)

// GenericImage is an interface that implements the common functions of Image and ContainerImage
type GenericImage interface {
	GetId() string
	GetName() *storage.ImageName
}

// ToImage converts a storage.ContainerImage to a storage.Image
func ToImage(ci *storage.ContainerImage) *storage.Image {
	return &storage.Image{
		Id:             ci.GetId(),
		Name:           ci.GetName(),
		Names:          []*storage.ImageName{ci.GetName()},
		NotPullable:    ci.GetNotPullable(),
		IsClusterLocal: ci.GetIsClusterLocal(),
	}
}

// ToImageV2 converts a storage.ContainerImage to a storage.ImageV2
func ToImageV2(ci *storage.ContainerImage) *storage.ImageV2 {
	return &storage.ImageV2{
		Id:             ci.GetIdV2(),
		Name:           ci.GetName(),
		NotPullable:    ci.GetNotPullable(),
		IsClusterLocal: ci.GetIsClusterLocal(),
	}
}

// ToContainerImage converts a storage.Image to a storage.ContainerImage
func ToContainerImage(ci *storage.Image) *storage.ContainerImage {
	res := &storage.ContainerImage{
		Id:          ci.GetId(),
		Name:        ci.GetName(),
		NotPullable: ci.GetNotPullable(),
	}
	if features.FlattenImageData.Enabled() && ci.GetId() != "" {
		res.IdV2 = uuid.NewV5FromNonUUIDs(ci.GetName().GetFullName(), ci.GetId()).String()
	}
	return res
}

// ToContainerImageV2 converts a storage.ImageV2 to a storage.ContainerImage
func ToContainerImageV2(ci *storage.ImageV2) *storage.ContainerImage {
	return &storage.ContainerImage{
		IdV2:        ci.GetId(),
		Name:        ci.GetName(),
		NotPullable: ci.GetNotPullable(),
	}
}

// ConvertImageToListImage converts an image to a ListImage
func ConvertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{
		Id:          i.GetId(),
		Name:        i.GetName().GetFullName(),
		Created:     i.GetMetadata().GetV1().GetCreated(),
		LastUpdated: i.GetLastUpdated(),
	}
	if i.GetSetComponents() != nil {
		listImage.SetComponents = &storage.ListImage_Components{
			Components: i.GetComponents(),
		}
	}
	if i.GetSetCves() != nil {
		listImage.SetCves = &storage.ListImage_Cves{
			Cves: i.GetCves(),
		}
	}
	if i.GetSetFixable() != nil {
		listImage.SetFixable = &storage.ListImage_FixableCves{
			FixableCves: i.GetFixableCves(),
		}
	}
	return listImage
}
