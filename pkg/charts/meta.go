package charts

import (
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
		"RenderMode":        "",
		"ChartRepo": ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		"ImagePullSecrets": ImagePullSecrets{
			AllowNone: false,
		},
	}

	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	metaValues["FeatureFlags"] = featureFlagVals

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
	}

	if !buildinfo.ReleaseBuild {
		metaValues["MainRegistry"] = "docker.io/stackrox"
		metaValues["CollectorRegistry"] = "docker.io/stackrox"
	}

	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	metaValues["FeatureFlags"] = featureFlagVals

	return metaValues
}
