package types

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// FromContainers provides helper functions for getting a slice of images from containers.
type FromContainers []*storage.Alert_Deployment_Container

// Images returns a slice of images from the slice of containers.
func (cs FromContainers) Images() []*storage.ContainerImage {
	images := make([]*storage.ContainerImage, len(cs))
	for i, c := range cs {
		images[i] = c.GetImage()
	}
	return images
}

func (cs FromContainers) String() string {
	output := make([]string, 0, len(cs))
	for _, c := range cs {
		output = append(output, Wrapper{GenericImage: c.GetImage()}.FullName())
	}
	return strings.Join(output, ", ")
}
