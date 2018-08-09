package types

import "github.com/stackrox/rox/generated/api/v1"

// FromContainers provides helper functions for getting a slice of images from containers.
type FromContainers []*v1.Container

// Images returns a slice of images from the slice of containers.
func (cs FromContainers) Images() []*v1.Image {
	images := make([]*v1.Image, len(cs))
	for i, c := range cs {
		images[i] = c.GetImage()
	}
	return images
}

func (cs FromContainers) String() string {
	return SliceWrapper(cs.Images()).String()
}
