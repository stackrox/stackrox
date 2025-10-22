//go:build test_e2e || test_compatibility

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKernelSupportAvailableApi(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewClustersServiceClient(conn)
	resp, err := service.GetKernelSupportAvailable(ctx, &v1.Empty{})

	// Central in CI is deployed in online mode, hence the expectation is
	// that kernel support is available via the HTTP download site.
	// Use require.NoError to stop test execution if the API call fails,
	// preventing nil pointer dereference on the response.
	require.NoError(t, err)
	assert.True(t, resp.GetKernelSupportAvailable())
}
