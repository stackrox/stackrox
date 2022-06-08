package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewPingServiceClient(conn)
	_, err := service.Ping(ctx, &v1.Empty{})
	assert.NoError(t, err)
}
