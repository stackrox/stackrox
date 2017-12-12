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

// ImageWrapper provides helper functions around the proto image type
type ImageWrapper struct {
	*v1.Image
}

func (i ImageWrapper) String() string {
	return fmt.Sprintf("%v/%v:%v", i.Registry, i.Remote, i.Tag)
}
