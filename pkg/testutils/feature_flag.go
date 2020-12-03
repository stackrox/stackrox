package testutils

import (
	"fmt"
	"testing"

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
