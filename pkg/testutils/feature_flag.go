package testutils

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
)

// RunWithAndWithoutFeatureFlag runs the given subtest both with and without the given feature flag set.
func RunWithAndWithoutFeatureFlag(t *testing.T, flagEnvVar, subTestName string, subTest func(t *testing.T)) {
	for _, value := range []string{"false", "true"} {
		t.Run(fmt.Sprintf("%s (with %s=%s)", subTestName, flagEnvVar, value), func(t *testing.T) {
			envIsolator := envisolator.NewEnvIsolator(t)
			defer envIsolator.RestoreAll()
			envIsolator.Setenv(flagEnvVar, value)
			subTest(t)
		})
	}
}

// RunWithFeatureFlagEnabled runs the given subtest if the feature can be enabled.
func RunWithFeatureFlagEnabled(t *testing.T, flag features.FeatureFlag, subTest func(t *testing.T)) {
	envIsolator := envisolator.NewEnvIsolator(t)
	defer envIsolator.RestoreAll()
	envIsolator.Setenv(flag.EnvVar(), "true")

	if !flag.Enabled() {
		t.Skipf("Skipping test because feature flag %q is not enabled", flag.Name())
	}

	subTest(t)
}
