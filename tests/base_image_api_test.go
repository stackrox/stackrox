//go:build test_e2e || test_compatibility

package tests

import (
	"context"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
)

func TestBaseImageServicePing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v2.NewBaseImageServiceV2Client(conn)

	_, err := service.GetBaseImageReferences(ctx, &v2.Empty{})

	require.NoError(t, err, "BaseImageServiceV2 should be reachable and implemented")
}
