package tests

import (
	"context"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataIsSetCorrectly(t *testing.T) {
	t.Parallel()

	if _, ok := os.LookupEnv("CI"); !ok {
		t.Skip("Skipping metadata test because we are not on CI")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := testutils.GRPCConnectionToCentral(t)

	service := v1.NewMetadataServiceClient(conn)
	metadata, err := service.GetMetadata(ctx, &v1.Empty{})
	require.NoError(t, err)
	if tag := os.Getenv("CIRCLE_TAG"); tag != "" {
		assert.Equal(t, "release", metadata.GetBuildFlavor())
		assert.True(t, metadata.ReleaseBuild)
		assert.Equal(t, tag, metadata.Version)
	} else {
		assert.Equal(t, "development", metadata.GetBuildFlavor())
		assert.False(t, metadata.ReleaseBuild)
	}

}
