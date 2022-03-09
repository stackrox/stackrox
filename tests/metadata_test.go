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
	"google.golang.org/grpc"
)

func getMetadata(t *testing.T, conn *grpc.ClientConn) *v1.Metadata {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	service := v1.NewMetadataServiceClient(conn)
	metadata, err := service.GetMetadata(ctx, &v1.Empty{})
	require.NoError(t, err)
	return metadata
}

func TestMetadataIsSetCorrectly(t *testing.T) {
	t.Parallel()

	if _, ok := os.LookupEnv("CI"); !ok {
		t.Skip("Skipping metadata test because we are not on CI")
		return
	}

	metadataWithAuth := getMetadata(t, testutils.GRPCConnectionToCentral(t))
	assert.Equal(t, buildinfo.BuildFlavor, metadataWithAuth.GetBuildFlavor())
	assert.Equal(t, buildinfo.ReleaseBuild, metadataWithAuth.GetReleaseBuild())
	assert.Equal(t, version.GetMainVersion(), metadataWithAuth.GetVersion())
	assert.Equal(t, v1.Metadata_RHACS_BRANDING, metadataWithAuth.GetProductBranding())

	// Test that an unauthenticated connection doesn't get the version.
	metadataWithoutAuth := getMetadata(t, testutils.UnauthenticatedGRPCConnectionToCentral(t))
	assert.Equal(t, buildinfo.BuildFlavor, metadataWithoutAuth.GetBuildFlavor())
	assert.Equal(t, buildinfo.ReleaseBuild, metadataWithoutAuth.GetReleaseBuild())
	assert.Equal(t, "", metadataWithoutAuth.GetVersion())
	assert.Equal(t, v1.Metadata_RHACS_BRANDING, metadataWithAuth.GetProductBranding())
}
