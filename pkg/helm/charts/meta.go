package charts

import (
	"fmt"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValuesKey exists exclusively to protect MetaValues from losing typing and becoming exchangeable with
// map[string]interface{}. By doing this we get the opportunity to more reliably trace MetaValues usage throughout the
// codebase.
// TODO(RS-379): Switch MetaValues to be struct and get rid of MetaValuesKey.
type MetaValuesKey string

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues map[MetaValuesKey]interface{}

// ChartRepo contains information about where the Helm charts are published.
type ChartRepo struct {
	URL string
}

// ImagePullSecrets represents the image pull secret defaults.
type ImagePullSecrets struct {
	AllowNone bool
}

// DefaultMetaValues are the default meta values for rendering the StackRox charts in production.
func DefaultMetaValues() MetaValues {
	metaValues := MetaValues{
		"Versions":              version.GetAllVersions(),
		"MainRegistry":          defaults.MainImageRegistry(),
		"ImageRemote":           "main",
		"CollectorRegistry":     defaults.CollectorImageRegistry(),
		"CollectorImageRemote":  "collector",
		"CollectorFullImageTag": fmt.Sprintf("%s-latest", version.GetCollectorVersion()),
		"CollectorSlimImageTag": fmt.Sprintf("%s-slim", version.GetCollectorVersion()),
		"RenderMode":            "",
		"ChartRepo": ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		"ImagePullSecrets": ImagePullSecrets{
			AllowNone: false,
		},
		"Operator": false,
	}

	if !buildinfo.ReleaseBuild {
		metaValues["FeatureFlags"] = getFeatureFlags()
	}

	return metaValues
}

// RHACSMetaValues are the meta values for rendering the StackRox charts in RHACS flavor.
func RHACSMetaValues() MetaValues {
	metaValues := MetaValues{
		"Versions":              version.GetAllVersions(),
		"MainRegistry":          "registry.redhat.io/rh-acs",
		"ImageRemote":           "main",
		"CollectorRegistry":     "registry.redhat.io/rh-acs",
		"CollectorImageRemote":  "collector",
		"CollectorFullImageTag": fmt.Sprintf("%s-latest", version.GetCollectorVersion()),
		"CollectorSlimImageTag": fmt.Sprintf("%s-slim", version.GetCollectorVersion()),
		"RenderMode":            "",
		"ChartRepo": ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},
		"ImagePullSecrets": ImagePullSecrets{
			AllowNone: true,
		},
		"Operator": false,
	}

	if !buildinfo.ReleaseBuild {
		// TODO(ROX-7740): Temporarily use images from quay until our private registries are up again
		metaValues["MainRegistry"] = mainRegistryOverride.Setting()
		metaValues["CollectorRegistry"] = collectorRegistryOverride.Setting()
		metaValues["FeatureFlags"] = getFeatureFlags()
	}

	return metaValues
}

// ToRaw converts MetaValues to map[string]interface{} for use in Go templating.
// Go templating does not like our MetaValuesKey and prefers to have string as a key in the map.
// Unfortunately, an attempt to cast MetaValues to map[string]interface{} does not compile, therefore we need to copy
// the map.
// TODO(RS-379): Switch MetaVals to struct and get rid of ToRaw function.
func (m MetaValues) ToRaw() map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[string(k)] = v
	}
	return result
}

func getFeatureFlags() map[string]interface{} {
	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	return featureFlagVals
}
