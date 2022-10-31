package testutils

import (
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
