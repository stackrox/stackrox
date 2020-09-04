package tests

import (
	"context"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
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
	assert.Equal(t, buildinfo.BuildFlavor, metadata.GetBuildFlavor())
	assert.Equal(t, buildinfo.ReleaseBuild, metadata.GetReleaseBuild())
	assert.Equal(t, version.GetMainVersion(), metadata.GetVersion())
}
