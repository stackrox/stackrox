package charts

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
)

// MetaValuesKey exists exclusively to protect MetaValues from losing typing and becoming exchangeable with
// map[string]interface{}. By doing this we get the opportunity to more reliably trace MetaValues usage throughout the
// codebase.
// TODO(RS-379): Switch MetaValues to be struct and get rid of MetaValuesKey.
type MetaValuesKey string

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues map[MetaValuesKey]interface{}

// GetMetaValuesForFlavor are the default meta values for rendering the StackRox charts in production.
func GetMetaValuesForFlavor(imageFlavor defaults.ImageFlavor) MetaValues {
	metaValues := MetaValues{
		"Versions":                 imageFlavor.Versions,
		"MainRegistry":             imageFlavor.MainRegistry,
		"ImageRemote":              imageFlavor.MainImageName,
		"ImageTag":                 imageFlavor.MainImageTag,
		"CollectorRegistry":        imageFlavor.CollectorRegistry,
		"CollectorFullImageRemote": imageFlavor.CollectorImageName,
		"CollectorSlimImageRemote": imageFlavor.CollectorSlimImageName,
		"CollectorFullImageTag":    imageFlavor.CollectorImageTag,
		"CollectorSlimImageTag":    imageFlavor.CollectorSlimImageTag,
		"ScannerImageRemote":       imageFlavor.ScannerImageName,
		"ScannerImageTag":          imageFlavor.ScannerImageTag,
		"ScannerDBImageRemote":     imageFlavor.ScannerDBImageName,
		"ScannerDBImageTag":        imageFlavor.ScannerDBImageTag,
		"RenderMode":               "",
		"ChartRepo":                imageFlavor.ChartRepo,
		"ImagePullSecrets":         imageFlavor.ImagePullSecrets,
		"Operator":                 false,
	}

	if !buildinfo.ReleaseBuild {
		metaValues["FeatureFlags"] = getFeatureFlags()
	}

	return metaValues
}

// RHACSMetaValues are the meta values for rendering the StackRox charts in RHACS flavor.
func RHACSMetaValues() MetaValues {
	// TODO(RS-380): remove once RHACS flavor is added to `images` package
	flavor := defaults.GetImageFlavorByBuildType()
	metaValues := MetaValues{
		"Versions": flavor.Versions,
		// TODO(RS-380): these registries will change once we have the RHACS flavor. For now they will remain hardcoded here.
		"MainRegistry":             "registry.redhat.io/rh-acs",
		"ImageRemote":              "main",
		"ImageTag":                 flavor.MainImageTag,
		"CollectorRegistry":        "registry.redhat.io/rh-acs",
		"CollectorFullImageRemote": "collector",
		"CollectorSlimImageRemote": "collector",
		"CollectorFullImageTag":    flavor.CollectorImageTag,
		"CollectorSlimImageTag":    flavor.CollectorSlimImageTag,
		"ScannerImageRemote":       flavor.ScannerImageName,
		"ScannerImageTag":          flavor.ScannerImageTag,
		"ScannerDBImageRemote":     flavor.ScannerDBImageName,
		"ScannerDBImageTag":        flavor.ScannerDBImageTag,
		"RenderMode":               "",
		"ChartRepo": defaults.ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},
		"ImagePullSecrets": defaults.ImagePullSecrets{
			AllowNone: true,
		},
		"Operator": false,
	}

	// TODO(RS-380): move or remove this block - this override is done only for the operator
	if !buildinfo.ReleaseBuild {
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
