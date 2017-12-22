package images

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/docker/distribution/reference"
)

// GenerateImageFromString generates an image type from a common string format
func GenerateImageFromString(imageStr string) *v1.Image {
	var image v1.Image

	// Check if its a sha and return if it is
	if strings.HasPrefix(imageStr, "sha256:") {
		image.Sha = strings.TrimPrefix(imageStr, "sha256:")
		return &image
	}

	// Cut off @sha256:
	if idx := strings.Index(imageStr, "@sha256:"); idx != -1 {
		image.Sha = imageStr[idx+len("@sha256:"):]
		imageStr = imageStr[:idx]
	}

	named, _ := reference.ParseNormalizedNamed(imageStr)
	tag := "latest"
	namedTagged, ok := named.(reference.NamedTagged)
	if ok {
		tag = namedTagged.Tag()
	}
	image.Remote = reference.Path(named)
	image.Tag = tag
	image.Registry = reference.Domain(named)
	return &image
}

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

// SliceWrapper provides helper functions for a slice of images.
type SliceWrapper []*v1.Image

func (s SliceWrapper) String() string {
	var output []string
	for _, img := range s {
		output = append(output, Wrapper{img}.String())
	}

	return strings.Join(output, ", ")
}

// Wrapper provides helper functions for an image.
type Wrapper struct {
	*v1.Image
}

func (i Wrapper) String() string {
	return fmt.Sprintf("%v/%v:%v", i.Registry, i.Remote, i.Tag)
}

// ShortID returns the SHA truncated to 12 characters
func (i Wrapper) ShortID() string {
	if len(i.Sha) <= 12 {
		return i.Sha
	}
	return i.Sha[:12]
}
