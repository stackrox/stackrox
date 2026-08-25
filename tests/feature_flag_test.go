//go:build test_e2e

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// skipFeatures is a list of feature flags to ignore in comparisons, these
	// flags may intentionally differ from the defaults.
	skipFeatures = map[string]bool{
		// Scanner V4 is enabled/disabled at install time, when Scanner V4 is
		// installed the value will differ from the default.
		features.ScannerV4.EnvVar(): true,
		// Legacy Scanner is enabled/disabled at install time, when the legacy
		// scanner is installed the value will differ from the default.
		features.LegacyScanner.EnvVar(): true,
	}
)

// TestFeatureFlagSettings asserts Central's live flags match flag.Enabled()
// computed in this process from the same environment and build type.
func TestFeatureFlagSettings(t *testing.T) {
	if os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift" {
		t.Skip("Skipping on OCP: env vars set by the test runner via ci_export are not reflected in the already-deployed Central, causing a systemic mismatch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	expectedFlagVals := make(map[string]bool)
	for _, flag := range features.Flags {
		if skipFeatures[flag.EnvVar()] {
			continue
		}

		expectedFlagVals[flag.EnvVar()] = flag.Enabled()
	}

	featureFlagService := v1.NewFeatureFlagServiceClient(conn)
	featureFlags, err := featureFlagService.GetFeatureFlags(ctx, &v1.Empty{})
	require.NoError(t, err, "failed to retrieve feature flags")

	actualFlagVals := make(map[string]bool)
	for _, flag := range featureFlags.GetFeatureFlags() {
		if skipFeatures[flag.GetEnvVar()] {
			continue
		}

		actualFlagVals[flag.GetEnvVar()] = flag.GetEnabled()
	}

	assert.Equal(t, expectedFlagVals, actualFlagVals, "mismatch between expected and actual feature flag settings")
}
