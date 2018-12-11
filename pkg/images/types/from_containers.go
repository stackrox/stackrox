package types

import (
	"github.com/stackrox/rox/generated/storage"
)

// FromContainers provides helper functions for getting a slice of images from containers.
type FromContainers []*storage.Container

// Images returns a slice of images from the slice of containers.
func (cs FromContainers) Images() []*storage.Image {
	images := make([]*storage.Image, len(cs))
	for i, c := range cs {
		images[i] = c.GetImage()
	}
	return images
}

func (cs FromContainers) String() string {
	return SliceWrapper(cs.Images()).String()
}
