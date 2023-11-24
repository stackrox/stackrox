package utils

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	defaultDockerRegistry = "docker.io"
)

var (
	// digestPrefixes lists the prefixes for valid, OCI-compliant image digests.
	// Please see https://github.com/opencontainers/image-spec/blob/main/descriptor.md#registered-algorithms
	// for more information.
	digestPrefixes = []string{"sha256:", "sha512:"}
)

// GenerateImageFromStringWithDefaultTag generates an image type from a common string format and returns an error if
// there was an issue parsing it. It takes in a defaultTag which it populates if the image doesn't have a tag.
func GenerateImageFromStringWithDefaultTag(imageStr, defaultTag string) (*storage.ContainerImage, error) {
	imageName, ref, err := GenerateImageNameFromString(imageStr)
	if err != nil {
		return nil, err
	}

	image := &storage.ContainerImage{
		Name:        imageName,
		NotPullable: false,
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
// issue parsing it.
func GenerateImageNameFromString(imageStr string) (*storage.ImageName, reference.Reference, error) {
	name := &storage.ImageName{
		FullName: imageStr,
	}

	ref, err := reference.ParseAnyReference(imageStr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing image name '%s'", imageStr)
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
func GenerateImageFromString(imageStr string) (*storage.ContainerImage, error) {
	return GenerateImageFromStringWithDefaultTag(imageStr, "latest")
}

// GenerateImageFromStringWithOverride will override the default value of docker.io if it was not specified in the full image name
// e.g. nginx:latest -> <registry override>/library/nginx;latest
func GenerateImageFromStringWithOverride(imageStr, registryOverride string) (*storage.ContainerImage, error) {
	image, err := GenerateImageFromString(imageStr)
	if err != nil {
		return nil, err
	}
	if registryOverride == "" {
		return image, err
	}

	// Only dockerhub can be mirrored: https://docs.docker.com/registry/recipes/mirror/
	if image.GetName().GetRegistry() == defaultDockerRegistry {
		image.Name.Registry = registryOverride

		trimmedFullName := strings.TrimPrefix(image.GetName().GetFullName(), defaultDockerRegistry)
		image.Name.FullName = fmt.Sprintf("%s%s", registryOverride, trimmedFullName)
	}
	return image, nil
}

// GetSHA returns the SHA of the image, if it exists.
func GetSHA(img *storage.Image) string {
	return stringutils.FirstNonEmpty(
		img.GetId(),
		img.GetMetadata().GetV2().GetDigest(),
		img.GetMetadata().GetV1().GetDigest(),
	)
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

// IsPullable returns whether Kubernetes thinks the image is pullable.
func IsPullable(imageStr string) bool {
	parts := strings.SplitN(imageStr, "://", 2)
	if len(parts) == 2 {
		if parts[0] == "docker-pullable" {
			return true
		}
		if parts[0] == "docker" {
			return false
		}
		imageStr = parts[1]
	}
	_, err := GenerateImageFromString(imageStr)
	return err == nil
}

// IsValidImageString returns whether the given string can be parsed as a docker image reference
func IsValidImageString(imageStr string) error {
	_, err := reference.ParseAnyReference(imageStr)
	return err
}

// ExtractImageDigest returns the image sha, if it exists, within the string.
// Otherwise, the empty string is returned.
func ExtractImageDigest(imageStr string) string {
	for _, prefix := range digestPrefixes {
		if idx := strings.Index(imageStr, prefix); idx != -1 {
			return imageStr[idx:]
		}
	}

	return ""
}

// ExtractOpenShiftProject returns the name of the OpenShift project in which the given image is stored.
// Images stored in the OpenShift Internal Registry are identified as: <registry>/<project>/<name>:<tag>.
func ExtractOpenShiftProject(imgName *storage.ImageName) string {
	// Use the image name's "remote" field, as it encapsulates <project>/<name>.
	return stringutils.GetUpTo(imgName.GetRemote(), "/")
}

type nameHolder interface {
	GetName() *storage.ImageName
	GetId() string
}

// GetFullyQualifiedFullName takes in an id and image name and returns the full name including sha256 if it exists
func GetFullyQualifiedFullName(holder nameHolder) string {
	if holder.GetId() == "" {
		return holder.GetName().GetFullName()
	}
	if idx := strings.Index(holder.GetName().GetFullName(), "@"); idx != -1 {
		return holder.GetName().GetFullName()
	}
	return fmt.Sprintf("%s@%s", holder.GetName().GetFullName(), holder.GetId())
}

// StripCVEDescriptions takes in an image and returns a stripped down version without the descriptions of CVEs
func StripCVEDescriptions(img *storage.Image) *storage.Image {
	newImage := img.Clone()
	StripCVEDescriptionsNoClone(newImage)
	return newImage
}

// StripCVEDescriptionsNoClone takes in an image object and modifies it to remove the vulnerability summaries
func StripCVEDescriptionsNoClone(img *storage.Image) {
	for _, component := range img.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.Summary = ""
		}
	}
}

// FilterSuppressedCVEsNoClone removes the vulns from the image that are currently suppressed
func FilterSuppressedCVEsNoClone(img *storage.Image) {
	cveSet := set.NewStringSet()
	for _, c := range img.GetScan().GetComponents() {
		filteredVulns := make([]*storage.EmbeddedVulnerability, 0, len(c.GetVulns()))
		for _, vuln := range c.GetVulns() {
			if !cve.IsCVESnoozed(vuln) {
				cveSet.Add(vuln.GetCve())
				filteredVulns = append(filteredVulns, vuln)
			}
		}
		c.Vulns = filteredVulns
	}
	if img.GetSetCves() != nil {
		img.SetCves = &storage.Image_Cves{
			Cves: int32(len(cveSet)),
		}
	}
}

// UniqueImageNames returns the unique image names from the two given slices of image names.
func UniqueImageNames(a, b []*storage.ImageName) []*storage.ImageName {
	uniqueImageNames := sliceutils.ShallowClone(a)
	for _, imageName := range b {
		if !protoutils.SliceContains(imageName, uniqueImageNames) {
			uniqueImageNames = append(uniqueImageNames, imageName)
		}
	}
	return uniqueImageNames
}
