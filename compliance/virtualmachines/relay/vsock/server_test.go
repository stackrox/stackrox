package vsock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestSemaphore(t *testing.T) {
	ctx := context.Background()

	vsockServer := &serverImpl{
		semaphore:        semaphore.NewWeighted(1),
		semaphoreTimeout: 5 * time.Millisecond,
	}

	// First should succeed
	err := vsockServer.AcquireSemaphore(ctx)
	require.NoError(t, err)

	// Second should time out
	err = vsockServer.AcquireSemaphore(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to acquire semaphore")

	// After releasing once, a new acquire should succeed
	vsockServer.ReleaseSemaphore()
	err = vsockServer.AcquireSemaphore(ctx)
	require.NoError(t, err)
}
