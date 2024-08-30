package testutils

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/features"
)

// MustUpdateFeature will attempt to set the feature flag to the desired value,
// if unable to do so will skip the test.
func MustUpdateFeature(t *testing.T, flag features.FeatureFlag, desired bool) {
	t.Setenv(flag.EnvVar(), strconv.FormatBool(desired))

	if flag.Enabled() != desired {
		t.Skipf("Skipping test, feature %q cannot be set to desired value: %t", flag.EnvVar(), desired)
	}
}
