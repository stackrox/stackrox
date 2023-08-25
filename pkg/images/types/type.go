package types

import "github.com/stackrox/rox/generated/storage"

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

// ToContainerImage converts a storage.Image to a storage.ContainerImage
func ToContainerImage(ci *storage.Image) *storage.ContainerImage {
	return &storage.ContainerImage{
		Id:          ci.GetId(),
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
