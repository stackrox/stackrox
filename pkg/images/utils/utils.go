package utils

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

// GenerateImageFromStringWithError generates an image type from a common string format and returns an error if
// there was an issue parsing it
func GenerateImageFromStringWithError(imageStr string) (*v1.Image, error) {
	image := &v1.Image{
		Name: &v1.ImageName{
			FullName: imageStr,
		},
	}

	ref, err := reference.ParseAnyReference(imageStr)
	if err != nil {
		return image, fmt.Errorf("error parsing image name '%s': %s", imageStr, err)
	}

	digest, ok := ref.(reference.Digested)
	if ok {
		image.Id = digest.Digest().String()
	}

	named, ok := ref.(reference.Named)
	if ok {
		image.Name.Registry = reference.Domain(named)
		image.Name.Remote = reference.Path(named)
	}

	namedTagged, ok := ref.(reference.NamedTagged)
	if ok {
		image.Name.Registry = reference.Domain(namedTagged)
		image.Name.Remote = reference.Path(namedTagged)
		image.Name.Tag = namedTagged.Tag()
	}

	// Default the image to latest if and only if there was no tag specific and also no SHA specified
	if image.GetId() == "" && image.GetName().GetTag() == "" {
		image.Name.Tag = "latest"
		image.Name.FullName = ref.String() + ":latest"
	} else {
		image.Name.FullName = ref.String()
	}

	return image, nil
}

// GenerateImageFromString generates an image type from a common string format
func GenerateImageFromString(imageStr string) *v1.Image {
	image, err := GenerateImageFromStringWithError(imageStr)
	if err != nil {
		logger.Error(err)
	}
	return image
}

// Reference returns what to use as the reference when talking to registries
func Reference(img *v1.Image) string {
	// If the image id is empty, then use the tag as the reference
	if img.GetId() != "" {
		return img.GetId()
	} else if img.GetName().GetTag() != "" {
		return img.GetName().GetTag()
	}
	return "latest"
}

// ExtractImageSha returns the image sha if it exists within the string.
func ExtractImageSha(imageStr string) string {
	if idx := strings.Index(imageStr, "sha256:"); idx != -1 {
		return imageStr[idx:]
	}

	return ""
}
