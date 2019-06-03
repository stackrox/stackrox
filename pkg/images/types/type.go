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
		Id:   ci.GetId(),
		Name: ci.GetName(),
	}
}

// ToContainerImage converts a storage.Image to a storage.ContainerImage
func ToContainerImage(ci *storage.Image) *storage.ContainerImage {
	return &storage.ContainerImage{
		Id:   ci.GetId(),
		Name: ci.GetName(),
	}
}
