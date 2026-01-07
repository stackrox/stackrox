package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/features"
)

// SetFlattenImageDataForTest sets the FlattenImageData feature flag for testing and returns a restore function.
// This may not be necessary if the environment is not persisted between tests.
func SetFlattenImageDataForTest(t *testing.T, enabled bool) func() {
	originalValue := features.FlattenImageData.Enabled()
	t.Setenv(features.FlattenImageData.EnvVar(), "false")
	if enabled {
		t.Setenv(features.FlattenImageData.EnvVar(), "true")
	}
	return func() {
		if originalValue {
			t.Setenv(features.FlattenImageData.EnvVar(), "true")
		} else {
			t.Setenv(features.FlattenImageData.EnvVar(), "false")
		}
	}
}
