package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureFlagSettings(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	metadataService := v1.NewMetadataServiceClient(conn)
	metadata, err := metadataService.GetMetadata(ctx, &v1.Empty{})
	require.NoError(t, err, "failed to retrieve metadata")

	expectedFlagVals := make(map[string]bool)
	for _, flag := range features.Flags {
		// For non-release builds, test that feature flag settings match the local environment;
		// for release builds, test that they match the defaults.
		expectedVal := flag.Enabled()
		if metadata.GetReleaseBuild() {
			expectedVal = flag.Default()
		}
		expectedFlagVals[flag.EnvVar()] = expectedVal
	}

	featureFlagService := v1.NewFeatureFlagServiceClient(conn)
	featureFlags, err := featureFlagService.GetFeatureFlags(ctx, &v1.Empty{})
	require.NoError(t, err, "failed to retrieve feature flags")

	actualFlagVals := make(map[string]bool)
	for _, flag := range featureFlags.GetFeatureFlags() {
		actualFlagVals[flag.GetEnvVar()] = flag.GetEnabled()
	}

	assert.Equal(t, expectedFlagVals, actualFlagVals, "mismatch between expected and actual feature flag settings")
}
