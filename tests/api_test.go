package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := testutils.GRPCConnectionToCentral(t)

	service := v1.NewPingServiceClient(conn)
	_, err := service.Ping(ctx, &v1.Empty{})
	assert.NoError(t, err)
}
