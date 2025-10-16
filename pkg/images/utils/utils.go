package utils

import (
	"fmt"
	"slices"
	"strings"

	"github.com/distribution/reference"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	defaultDockerRegistry = "docker.io"
)

var (
	// digestPrefixes lists the prefixes for valid, OCI-compliant image digests.
	// Please see https://github.com/opencontainers/image-spec/blob/main/descriptor.md#registered-algorithms
	// for more information.
	digestPrefixes = []string{"sha256:", "sha512:"}

	// redHatRegistries contains registries where all images are built by Red Hat.
	// See https://github.com/stackrox/stackrox/pull/15761 for details.
	redHatRegistries = set.NewFrozenStringSet(
		"registry.access.redhat.com",
		"registry.redhat.io",
	)

	// quayIoRedHatRemotes contains quay.io remotes where all images are built by Red Hat.
	// See https://github.com/stackrox/stackrox/pull/15761 for details.
	quayIoRedHatRemotes = set.NewFrozenStringSet(
		"openshift-release-dev/ocp-release",
		"openshift-release-dev/ocp-v4.0-art-dev",
	)
)

// GenerateImageFromStringWithDefaultTag generates an image type from a common string format and returns an error if
// there was an issue parsing it. It takes in a defaultTag which it populates if the image doesn't have a tag.
func GenerateImageFromStringWithDefaultTag(imageStr, defaultTag string) (*storage.ContainerImage, error) {
	imageName, ref, err := GenerateImageNameFromString(imageStr)
	if err != nil {
		return nil, err
	}

	image := &storage.ContainerImage{}
	image.SetName(imageName)
	image.SetNotPullable(false)

	digest, ok := ref.(reference.Digested)
	if ok {
		image.SetId(digest.Digest().String())
	}

	if features.FlattenImageData.Enabled() && image.GetId() != "" {
		image.SetIdV2(NewImageV2ID(image.GetName(), image.GetId()))
	}

	// Default the image to latest if and only if there was no tag specific and also no SHA specified
	if image.GetId() == "" && image.GetName().GetTag() == "" && defaultTag != "" {
		SetImageTagNoSha(image.GetName(), defaultTag)
	}

	return image, nil
}

// GenerateImageNameFromString generated an ImageName from a common string format and returns an error if there was an
// issue parsing it.
func GenerateImageNameFromString(imageStr string) (*storage.ImageName, reference.Reference, error) {
	name := &storage.ImageName{}
	name.SetFullName(imageStr)

	ref, err := reference.ParseAnyReference(imageStr)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing image name '%s'", imageStr)
	}

	named, ok := ref.(reference.Named)
	if ok {
		name.SetRegistry(reference.Domain(named))
		name.SetRemote(reference.Path(named))
	}

	namedTagged, ok := ref.(reference.NamedTagged)
	if ok {
		name.SetRegistry(reference.Domain(namedTagged))
		name.SetRemote(reference.Path(namedTagged))
		name.SetTag(namedTagged.Tag())
	}

	name.SetFullName(ref.String())

	return name, ref, nil
}

// SetImageTagNoSha sets the tag on an ImageName and updates the FullName to reflect the new tag.  This function should be
// part of a wrapper instead of a util function
func SetImageTagNoSha(name *storage.ImageName, tag string) *storage.ImageName {
	name.SetTag(tag)
	NormalizeImageFullNameNoSha(name)
	return name
}

// NormalizeImageFullNameNoSha sets the ImageName.FullName correctly based on the parts of the name and should be part of a
// wrapper instead of a util function.
func NormalizeImageFullNameNoSha(name *storage.ImageName) *storage.ImageName {
	name.SetFullName(fmt.Sprintf("%s/%s:%s", name.GetRegistry(), name.GetRemote(), name.GetTag()))
	return name
}

// NormalizeImageFullName mimics NormalizeImageFullNameNoSha but accepts a digest,
// allows an empty tag, and does not modify name if it's malformed.
func NormalizeImageFullName(name *storage.ImageName, digest string) *storage.ImageName {
	if name.GetTag() == "" && digest == "" {
		// Input is malformed, do nothing.
		return name
	}

	if digest != "" {
		digest = fmt.Sprintf("@%s", digest)
	}

	tag := name.GetTag()
	if tag != "" {
		tag = fmt.Sprintf(":%s", tag)
	}

	name.SetFullName(fmt.Sprintf("%s/%s%s%s", name.GetRegistry(), name.GetRemote(), tag, digest))
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
		image.GetName().SetRegistry(registryOverride)

		trimmedFullName := strings.TrimPrefix(image.GetName().GetFullName(), defaultDockerRegistry)
		image.GetName().SetFullName(fmt.Sprintf("%s%s", registryOverride, trimmedFullName))
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

// GetSHAV2 returns the SHA of the imageV2, if it exists.
func GetSHAV2(img *storage.ImageV2) string {
	return stringutils.FirstNonEmpty(
		img.GetDigest(),
		img.GetMetadata().GetV2().GetDigest(),
		img.GetMetadata().GetV1().GetDigest(),
	)
}

// GetImageV2ID returns the ID of the imageV2, if it exists.
// If it does not exist, it returns the UUID V5 generated by combining image fullname and digest.
// It checks if the ID matches the expected UUID V5 generated by combining image fullname and digest.
func GetImageV2ID(img *storage.ImageV2) (string, error) {
	if img.GetId() != "" {
		expectedID := NewImageV2ID(img.GetName(), GetSHAV2(img))
		if img.GetId() != expectedID {
			return "", errors.Errorf("image ID '%s' does not match expected UUID V5 '%s' generated by combining image fullname and digest", img.GetId(), expectedID)
		}
		return img.GetId(), nil
	}
	return NewImageV2ID(img.GetName(), GetSHAV2(img)), nil
}

// NewImageV2ID generates a UUID V5 by combining image fullname and digest.
func NewImageV2ID(name *storage.ImageName, digest string) string {
	if digest == "" {
		return ""
	}
	if name.GetFullName() == "" {
		return ""
	}
	return uuid.NewV5FromNonUUIDs(name.GetFullName(), digest).String()
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

// RemoveScheme removes the scheme from an image string. For example:
// "docker-pullable://rest-of-image" becomes "rest-of-image"
func RemoveScheme(imageStr string) string {
	_, after, found := strings.Cut(imageStr, "://")
	if found {
		return after
	}
	return imageStr
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
	newImage := img.CloneVT()
	StripCVEDescriptionsNoClone(newImage)
	return newImage
}

// StripCVEDescriptionsNoClone takes in an image object and modifies it to remove the vulnerability summaries
func StripCVEDescriptionsNoClone(img *storage.Image) {
	for _, component := range img.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.SetSummary("")
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
		c.SetVulns(filteredVulns)
	}
	if img.GetSetCves() != nil {
		img.Set_Cves(int32(len(cveSet)))
	}
}

// IsRedHatImage takes in an image and returns whether it's a Red Hat image.
//
// This function is used to determine whether an image is supposed to have been built and signed by Red Hat, for supply
// chain provenance
func IsRedHatImage(img *storage.Image) bool {
	return slices.ContainsFunc(img.GetNames(), IsRedHatImageName)
}

// IsRedHatImage takes in an image and returns whether it's a Red Hat image.
//
// This function is used to determine whether an image is supposed to have been built and signed by Red Hat, for supply
// chain provenance
func IsRedHatImageV2(img *storage.ImageV2) bool {
	return IsRedHatImageName(img.GetName())
}

// IsRedHatImageName takes in an image name and returns whether it corresponds to a Red Hat image
//
// This is determined via heuristics, by looking at these images, which are assumed to be "official Red Hat images",
// and checking where they are hosted:
//
//   - All images running in a default openshift cluster
//   - All images that are referred by PackageManifests in the "redhat-operators" OLM catalog
//
// See the description of https://github.com/stackrox/stackrox/pull/15761 for details
func IsRedHatImageName(imgName *storage.ImageName) bool {
	// First consider registries where all images are built by Red Hat
	imageRegistry := imgName.GetRegistry()
	if redHatRegistries.Contains(imageRegistry) {
		return true
	}

	// The only remaining possibility is quay.io, where certain remotes are all Red Hat
	if imageRegistry != "quay.io" {
		return false
	}

	return quayIoRedHatRemotes.Contains(imgName.GetRemote())
}

type cveStats struct {
	fixable  bool
	severity storage.VulnerabilitySeverity
}

// FillScanStatsV2 fills in the higher level stats from the scan data.
func FillScanStatsV2(i *storage.ImageV2) {
	if i.GetScan() == nil {
		return
	}
	if i.GetScanStats() != nil {
		return
	}
	i.SetScanStats(&storage.ImageV2_ScanStats{})
	i.GetScanStats().SetComponentCount(int32(len(i.GetScan().GetComponents())))

	var imageTopCVSS float32
	vulns := make(map[string]*cveStats)

	// This enriches the incoming component.  When enriching any additional component fields,
	// be sure to update `ComponentIDV2` to ensure enriched fields like `TopCVSS` are not
	// included in the hash calculation
	for _, c := range i.GetScan().GetComponents() {
		var componentTopCVSS float32
		var hasVulns bool
		for _, v := range c.GetVulns() {
			hasVulns = true
			if _, ok := vulns[v.GetCve()]; !ok {
				vulns[v.GetCve()] = &cveStats{
					fixable:  false,
					severity: v.GetSeverity(),
				}
			}

			if v.GetCvss() > componentTopCVSS {
				componentTopCVSS = v.GetCvss()
			}

			if v.GetSetFixedBy() == nil {
				continue
			}

			if v.GetFixedBy() != "" {
				vulns[v.GetCve()].fixable = true
			}
		}

		if hasVulns {
			c.Set_TopCvss(componentTopCVSS)
		}

		if componentTopCVSS > imageTopCVSS {
			imageTopCVSS = componentTopCVSS
		}
	}

	i.GetScanStats().SetCveCount(int32(len(vulns)))
	i.SetTopCvss(imageTopCVSS)

	for _, vuln := range vulns {
		if vuln.fixable {
			i.GetScanStats().SetFixableCveCount(i.GetScanStats().GetFixableCveCount() + 1)
		}
		switch vuln.severity {
		case storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY:
			i.GetScanStats().SetUnknownCveCount(i.GetScanStats().GetUnknownCveCount() + 1)
			if vuln.fixable {
				i.GetScanStats().SetFixableUnknownCveCount(i.GetScanStats().GetFixableUnknownCveCount() + 1)
			}
		case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
			i.GetScanStats().SetCriticalCveCount(i.GetScanStats().GetCriticalCveCount() + 1)
			if vuln.fixable {
				i.GetScanStats().SetFixableCriticalCveCount(i.GetScanStats().GetFixableCriticalCveCount() + 1)
			}
		case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
			i.GetScanStats().SetImportantCveCount(i.GetScanStats().GetImportantCveCount() + 1)
			if vuln.fixable {
				i.GetScanStats().SetFixableImportantCveCount(i.GetScanStats().GetFixableImportantCveCount() + 1)
			}
		case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
			i.GetScanStats().SetModerateCveCount(i.GetScanStats().GetModerateCveCount() + 1)
			if vuln.fixable {
				i.GetScanStats().SetFixableModerateCveCount(i.GetScanStats().GetFixableModerateCveCount() + 1)
			}
		case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
			i.GetScanStats().SetLowCveCount(i.GetScanStats().GetLowCveCount() + 1)
			if vuln.fixable {
				i.GetScanStats().SetFixableLowCveCount(i.GetScanStats().GetFixableLowCveCount() + 1)
			}
		}
	}
}
