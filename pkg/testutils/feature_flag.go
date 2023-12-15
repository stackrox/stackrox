package testutils

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/features"
)

// RunWithFeatureFlagEnabled runs the given subtest if the feature can be enabled.
func RunWithFeatureFlagEnabled(t *testing.T, flag features.FeatureFlag, subTest func(t *testing.T)) {
	t.Setenv(flag.EnvVar(), "true")

	if !flag.Enabled() {
		t.Skipf("Skipping test because feature flag %q is not enabled", flag.Name())
	}

	subTest(t)
}

// MustUpdateFeature will attempt to set the feature flag to the desired value,
// if unable to do so will skip the the test.
func MustUpdateFeature(t *testing.T, flag features.FeatureFlag, desired bool) {
	t.Setenv(flag.EnvVar(), strconv.FormatBool(desired))

	if flag.Enabled() != desired {
		t.Skipf("Skipping test, feature %q cannot be set to desired value: %t", flag.EnvVar(), desired)
	}
}
