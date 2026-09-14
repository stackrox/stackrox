//go:build test_e2e

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getMetadata(t *testing.T, conn *grpc.ClientConn) *v1.Metadata {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	service := v1.NewMetadataServiceClient(conn)
	resp, err := service.GetMetadata(ctx, &v1.Empty{})
	require.NoError(t, err)
	return resp
}

func TestCentralVersionHeader(t *testing.T) {
	// Requires version stamped via ldflags (-X github.com/stackrox/rox/pkg/version/internal.MainVersion=<version>).
	// scripts/go-test.sh sets this automatically; for manual runs pass the flag or set MAIN_IMAGE_TAG.
	if version.GetMainVersion() == "" {
		t.Skip("Skipping: version not stamped (run via scripts/go-test.sh or pass -ldflags)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	service := v1.NewMetadataServiceClient(centralgrpc.GRPCConnectionToCentral(t))

	var md metadata.MD
	_, err := service.GetMetadata(ctx, &v1.Empty{}, grpc.Header(&md))
	require.NoError(t, err)

	vals := md.Get(clientconn.CentralVersionHeader)
	require.Len(t, vals, 1, "expected %s response header", clientconn.CentralVersionHeader)
	assert.Equal(t, version.GetMainVersion(), vals[0])
}

func TestCentralVersionHeader_AbsentForAnonymous(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	service := v1.NewMetadataServiceClient(centralgrpc.UnauthenticatedGRPCConnectionToCentral(t))

	var md metadata.MD
	_, _ = service.GetMetadata(ctx, &v1.Empty{}, grpc.Header(&md))

	vals := md.Get(clientconn.CentralVersionHeader)
	assert.Empty(t, vals, "anonymous requests must not receive %s", clientconn.CentralVersionHeader)
}

func TestMetadataIsSetCorrectly(t *testing.T) {
	if _, ok := os.LookupEnv("CI"); !ok {
		t.Skip("Skipping metadata test because we are not on CI")
		return
	}

	metadataWithAuth := getMetadata(t, centralgrpc.GRPCConnectionToCentral(t))
	assert.Equal(t, buildinfo.BuildFlavor, metadataWithAuth.GetBuildFlavor())
	assert.Equal(t, buildinfo.ReleaseBuild, metadataWithAuth.GetReleaseBuild())
	assert.Equal(t, version.GetMainVersion(), metadataWithAuth.GetVersion())
}
