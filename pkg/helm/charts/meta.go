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
		"ScannerSlimImageRemote":   imageFlavor.ScannerSlimImageName,
		"ScannerImageTag":          imageFlavor.ScannerImageTag,
		"ScannerDBImageRemote":     imageFlavor.ScannerDBImageName,
		"ScannerDBSlimImageRemote": imageFlavor.ScannerDBSlimImageName,
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
