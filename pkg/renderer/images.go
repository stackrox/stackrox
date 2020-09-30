package renderer

import (
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/version"
)

// computeImageOverrides takes in a full image reference as well as default registries, names,
// and tags, and computes the components of the image which are different. I.e., if
// `fullImageRef` is `<defRegistry>/<defName>:<defTag>`, an empty map is returned; if, for
// example, only the tag is different, a map containing only the non-default "Tag" is returned
// etc.
func computeImageOverrides(fullImageRef, defRegistry, defName, defTag string) map[string]string {
	overrides := map[string]string{}

	// See the goal of the override computation explained in the `configureImageOverrides`
	// comment below. This somewhat creative approach is the reason why we are not using
	// the existing image ref parsing functions.
	remoteAndRepo, tag := stringutils.Split2(fullImageRef, ":")
	if tag == "" {
		tag = "latest"
	}
	if tag != defTag {
		overrides["Tag"] = tag
	}
	if stringutils.ConsumeSuffix(&remoteAndRepo, "/"+defName) {
		if remoteAndRepo != defRegistry {
			overrides["Registry"] = remoteAndRepo
		}
	} else if stringutils.ConsumePrefix(&remoteAndRepo, defRegistry+"/") {
		overrides["Name"] = remoteAndRepo
	} else {
		registry, name := stringutils.Split2Last(remoteAndRepo, "/")
		overrides["Registry"] = registry
		overrides["Name"] = name
	}

	if len(overrides) == 0 {
		return nil
	}
	return overrides
}

// configureImageOverrides sets the `c.K8sConfig.ImageOverrides` property based on the actually
// configured images as well as the default image values.
// The terms "registry" and "image name" in the configuration are used in a less than strict
// fashion, and the goal is to arrive at a configuration that appears natural and minimizes
// repetitions.
// For example, if the central and scanner images are `docker.io/stackrox/main` and
// `docker.io/stackrox/scanner`, the inferred "registry" is `docker.io/stackrox`, and no image
// name overrides need to be inferred. However, if the images are `us.gcr.io/stackrox-main/my-main`
// and `us.gcr.io/stackrox-scanner/my-scanner`, the "registry" is `us.gcr.io`, and the image name
// overrides are `stackrox-main/my-main` and `stackrox-scanner/my-scanner` respectively (since the
// names have to be overridden anyway). If, on the other hand, the images are
// `us.gcr.io/stackrox-main/main` and `us.gcr.io/stackrox-scanner/scanner`, no name overrides are
// inferred, and instead the inferred central and scanner "registries" are
// `us.gcr.io/stackrox-main` and `us.gcr.io/stackrox-scanner`.
func configureImageOverrides(c *Config) {
	imageOverrides := make(map[string]interface{})

	mainOverrides := computeImageOverrides(c.K8sConfig.MainImage, defaults.MainImageRegistry(), "main", version.GetMainVersion())
	registry := mainOverrides["Registry"]
	if registry == "" {
		registry = defaults.MainImageRegistry()
	} else {
		imageOverrides["MainRegistry"] = registry
		delete(mainOverrides, "Registry")
	}
	imageOverrides["Main"] = mainOverrides

	imageOverrides["Scanner"] = computeImageOverrides(c.K8sConfig.ScannerImage, registry, "scanner", version.GetScannerVersion())
	imageOverrides["ScannerDB"] = computeImageOverrides(c.K8sConfig.ScannerDBImage, registry, "scanner-db", version.GetScannerVersion())

	c.K8sConfig.ImageOverrides = imageOverrides
}
