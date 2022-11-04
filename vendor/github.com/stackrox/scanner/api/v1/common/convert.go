package common

import (
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurefmt"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// CreateExecutablesFromDependencies creates an array of v1.Executable from a feature and its dependencies.
func CreateExecutablesFromDependencies(featureKey featurefmt.PackageKey, executableToDependencies database.StringToStringsMap, depMap map[string]FeatureKeySet) []*scannerV1.Executable {
	executables := make([]*scannerV1.Executable, 0, len(executableToDependencies))
	for exec, libs := range executableToDependencies {
		features := make(FeatureKeySet)
		features.Add(featureKey)
		for lib := range libs {
			features.Merge(depMap[lib])
		}
		executables = append(executables, &scannerV1.Executable{
			Path:             exec,
			RequiredFeatures: toFeatureNameVersions(features),
		})
	}
	return executables
}

func toFeatureNameVersions(keys FeatureKeySet) []*scannerV1.FeatureNameVersion {
	if len(keys) == 0 {
		return nil
	}
	features := make([]*scannerV1.FeatureNameVersion, 0, len(keys))
	for k := range keys {
		features = append(features, &scannerV1.FeatureNameVersion{Name: k.Name, Version: k.Version})
	}
	return features
}
