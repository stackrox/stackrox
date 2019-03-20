package utils

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

// GenerateImageFromStringWithDefaultTag generates an image type from a common string format and returns an error if
// there was an issue parsing it. It takes in a defaultTag which it populates if the image doesn't have a tag.
func GenerateImageFromStringWithDefaultTag(imageStr, defaultTag string) (*storage.Image, error) {
	image := &storage.Image{
		Name: &storage.ImageName{
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
	if image.GetId() == "" && image.GetName().GetTag() == "" && defaultTag != "" {
		image.Name.Tag = defaultTag
		image.Name.FullName = fmt.Sprintf("%s:%s", ref.String(), defaultTag)
	} else {
		image.Name.FullName = ref.String()
	}

	return image, nil
}

// GenerateImageFromString generates an image type from a common string format and returns an error if
// there was an issue parsing it
func GenerateImageFromString(imageStr string) (*storage.Image, error) {
	return GenerateImageFromStringWithDefaultTag(imageStr, "latest")
}

// GetSHA returns the SHA of the image if it exists
func GetSHA(img *storage.Image) string {
	if img.GetId() != "" {
		return img.GetId()
	}
	if d := img.GetMetadata().GetV2().GetDigest(); d != "" {
		return d
	}
	if d := img.GetMetadata().GetV1().GetDigest(); d != "" {
		return d
	}
	return ""
}

// Reference returns what to use as the reference when talking to registries
func Reference(img *storage.Image) string {
	// If the image id is empty, then use the tag as the reference
	if img.GetId() != "" {
		return img.GetId()
	} else if img.GetName().GetTag() != "" {
		return img.GetName().GetTag()
	}
	return "latest"
}

// GenerateImageFromStringIgnoringError generates an image type from a common string format
func GenerateImageFromStringIgnoringError(imageStr string) *storage.Image {
	image, err := GenerateImageFromString(imageStr)
	if err != nil {
		logger.Error(err)
	}
	return image
}

// ExtractImageSha returns the image sha if it exists within the string.
func ExtractImageSha(imageStr string) string {
	if idx := strings.Index(imageStr, "sha256:"); idx != -1 {
		return imageStr[idx:]
	}

	return ""
}
