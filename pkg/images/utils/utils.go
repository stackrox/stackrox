package utils

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// GenerateImageFromStringWithDefaultTag generates an image type from a common string format and returns an error if
// there was an issue parsing it. It takes in a defaultTag which it populates if the image doesn't have a tag.
func GenerateImageFromStringWithDefaultTag(imageStr, defaultTag string) (*storage.Image, error) {
	imageName, ref, err := GenerateImageNameFromString(imageStr)
	if err != nil {
		return nil, err
	}

	image := &storage.Image{
		Name: imageName,
	}

	digest, ok := ref.(reference.Digested)
	if ok {
		image.Id = digest.Digest().String()
	}

	// Default the image to latest if and only if there was no tag specific and also no SHA specified
	if image.GetId() == "" && image.GetName().GetTag() == "" && defaultTag != "" {
		SetImageTagNoSha(image.Name, defaultTag)
	}

	return image, nil
}

// GenerateImageNameFromString generated an ImageName from a common string format and returns an error if there was an
// issure parsing it.
func GenerateImageNameFromString(imageStr string) (*storage.ImageName, reference.Reference, error) {
	name := &storage.ImageName{
		FullName: imageStr,
	}

	ref, err := reference.ParseAnyReference(imageStr)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing image name '%s': %s", imageStr, err)
	}

	named, ok := ref.(reference.Named)
	if ok {
		name.Registry = reference.Domain(named)
		name.Remote = reference.Path(named)
	}

	namedTagged, ok := ref.(reference.NamedTagged)
	if ok {
		name.Registry = reference.Domain(namedTagged)
		name.Remote = reference.Path(namedTagged)
		name.Tag = namedTagged.Tag()
	}

	name.FullName = ref.String()

	return name, ref, nil
}

// SetImageTagNoSha sets the tag on an ImageName and updates the FullName to reflect the new tag.  This function should be
// part of a wrapper instead of a util function
func SetImageTagNoSha(name *storage.ImageName, tag string) *storage.ImageName {
	name.Tag = tag
	NormalizeImageFullNameNoSha(name)
	return name
}

// NormalizeImageFullNameNoSha sets the ImageName.FullName correctly based on the parts of the name and should be part of a
// wrapper instead of a util function.
func NormalizeImageFullNameNoSha(name *storage.ImageName) *storage.ImageName {
	name.FullName = fmt.Sprintf("%s/%s:%s", name.GetRegistry(), name.GetRemote(), name.GetTag())
	return name
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
		log.Error(err)
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
