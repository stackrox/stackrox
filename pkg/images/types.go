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
	image.Name.Registry = reference.Domain(named)
	image.Name.Remote = reference.Path(named)
	image.Name.Tag = tag
	image.Name.FullName = fmt.Sprintf("%s/%s:%s", image.Name.Registry, image.Name.Remote, image.Name.Tag)
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
		output[i] = img.GetName().GetFullName()
	}

	return strings.Join(output, ", ")
}

// Wrapper provides helper functions for an image.
type Wrapper struct {
	*v1.Image
}

// ShortID returns the SHA truncated to 12 characters.
func (i Wrapper) ShortID() string {
	withoutAlgorithm := NewDigest(i.GetName().GetSha()).Hash()
	if len(withoutAlgorithm) <= 12 {
		return withoutAlgorithm
	}
	return withoutAlgorithm[:12]
}

// Digest is a wrapper around a SHA so we can access it with or without a prefix
type Digest struct {
	algorithm string
	hash      string
}

// NewDigest returns an internal representation of a SHA.
func NewDigest(sha string) *Digest {
	var hash, algorithm string
	if idx := strings.Index(sha, ":"); idx != -1 {
		algorithm = sha[:idx]
		hash = sha[idx+1:]
	} else {
		algorithm = "sha256"
		hash = sha
	}
	return &Digest{
		algorithm: algorithm,
		hash:      hash,
	}
}

// Algorithm returns the algorithm used in the Digest
func (d Digest) Algorithm() string {
	return d.algorithm + ":" + d.hash
}

// Digest returns the entire Digest
func (d Digest) Digest() string {
	return d.algorithm + ":" + d.hash
}

// Hash returns the SHA without the sha256: prefix.
func (d Digest) Hash() string {
	return d.hash
}
