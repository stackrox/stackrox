package charts

import (
	"fmt"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues map[string]interface{}

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
	metaValues := map[string]interface{}{
		"Versions":          version.GetAllVersions(),
		"MainRegistry":      defaults.MainImageRegistry(),
		"CollectorRegistry": defaults.CollectorImageRegistry(),
		"CollectorImageTag": fmt.Sprintf("%s-latest", version.GetCollectorVersion()),
		"CollectorSlimImageTag": fmt.Sprintf("%s-slim", version.GetCollectorVersion()),
		"RenderMode":        "",
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
	metaValues := map[string]interface{}{
		"Versions":          version.GetAllVersions(),
		"MainRegistry":      "registry.redhat.io/rh-acs",
		"CollectorRegistry": "registry.redhat.io/rh-acs",
		"RenderMode":        "",
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

func getFeatureFlags() map[string]interface{} {
	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	return featureFlagVals
}
