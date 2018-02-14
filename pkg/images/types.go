package images

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/docker/distribution/reference"
)

// GenerateImageFromString generates an image type from a common string format
func GenerateImageFromString(imageStr string) *v1.Image {
	image := v1.Image{
		Name: &v1.ImageName{},
	}

	// Check if its a sha and return if it is
	if strings.HasPrefix(imageStr, "sha256:") {
		image.Name.Sha = strings.TrimPrefix(imageStr, "sha256:")
		return &image
	}

	// Cut off @sha256:
	if idx := strings.Index(imageStr, "@sha256:"); idx != -1 {
		image.Name.Sha = imageStr[idx+len("@sha256:"):]
		imageStr = imageStr[:idx]
	}

	named, err := reference.ParseNormalizedNamed(imageStr)
	if err != nil {
		return &image
	}
	tag := "latest"
	namedTagged, ok := named.(reference.NamedTagged)
	if ok {
		tag = namedTagged.Tag()
	}
	image.Name.Remote = reference.Path(named)
	image.Name.Tag = tag
	image.Name.Registry = reference.Domain(named)
	return &image
}

// ExtractImageSha returns the image sha if it exists within the string.
func ExtractImageSha(imageStr string) string {
	if idx := strings.Index(imageStr, "@sha256:"); idx != -1 {
		return imageStr[idx+len("@sha256:"):]
	}

	return ""
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
	output := make([]string, len(s))
	for i, img := range s {
		output[i] = Wrapper{img}.String()
	}

	return strings.Join(output, ", ")
}

// Wrapper provides helper functions for an image.
type Wrapper struct {
	*v1.Image
}

func (i Wrapper) String() string {
	return fmt.Sprintf("%v/%v:%v", i.GetName().GetRegistry(), i.GetName().GetRemote(), i.GetName().GetTag())
}

// GetSHA returns the trimmed sha of the image
func (i Wrapper) GetSHA() string {
	return strings.TrimPrefix(i.GetName().GetSha(), "sha256:")
}

// GetPrefixedSHA returns the SHA prefixed with sha256:
func (i Wrapper) GetPrefixedSHA() string {
	if strings.HasPrefix(i.GetName().GetSha(), "sha256:") {
		return i.GetName().GetSha()
	}
	return "sha256:" + i.GetName().GetSha()
}

// ShortID returns the SHA truncated to 12 characters.
func (i Wrapper) ShortID() string {
	sha := strings.TrimPrefix(i.GetName().GetSha(), "sha256:")

	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}
