package common

import (
	"github.com/stackrox/scanner/ext/featurefmt"
)

// FeatureKeySet contains a set of feature keys
type FeatureKeySet map[featurefmt.PackageKey]struct{}

// Merge adds all feature keys from the other feature set
func (f *FeatureKeySet) Merge(other FeatureKeySet) {
	if *f == nil && len(other) > 0 {
		*f = make(FeatureKeySet, len(other))
	}
	for key := range other {
		(*f)[key] = struct{}{}
	}
}

// Add adds a feature key to this feature key set.
func (f *FeatureKeySet) Add(featureKey featurefmt.PackageKey) {
	if *f == nil {
		*f = make(FeatureKeySet)
	}
	(*f)[featureKey] = struct{}{}
}
