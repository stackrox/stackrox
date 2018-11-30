package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewPingServiceClient(conn)
	_, err = service.Ping(ctx, &v1.Empty{})
	assert.NoError(t, err)
}
