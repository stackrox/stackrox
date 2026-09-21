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
	conn := centralgrpc.GRPCConnectionToCentral(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var metadataMD metadata.MD
	resp, err := v1.NewMetadataServiceClient(conn).GetMetadata(ctx, &v1.Empty{}, grpc.Header(&metadataMD))
	require.NoError(t, err)

	metadataVals := metadataMD.Get(clientconn.CentralVersionHeader)
	require.Len(t, metadataVals, 1, "expected %s response header from GetMetadata", clientconn.CentralVersionHeader)
	assert.Equal(t, resp.GetVersion(), metadataVals[0])

	var pingMD metadata.MD
	_, err = v1.NewPingServiceClient(conn).Ping(ctx, &v1.Empty{}, grpc.Header(&pingMD))
	require.NoError(t, err)

	pingVals := pingMD.Get(clientconn.CentralVersionHeader)
	require.Len(t, pingVals, 1, "expected %s response header from Ping", clientconn.CentralVersionHeader)
	assert.Equal(t, metadataVals[0], pingVals[0], "version header must be consistent across services")
}

func TestCentralVersionHeader_AbsentForAnonymous(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	service := v1.NewPingServiceClient(centralgrpc.UnauthenticatedGRPCConnectionToCentral(t))

	var md metadata.MD
	_, err := service.Ping(ctx, &v1.Empty{}, grpc.Header(&md))
	require.NoError(t, err)

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
