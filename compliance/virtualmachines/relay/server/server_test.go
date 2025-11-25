package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestSemaphore(t *testing.T) {
	ctx := context.Background()

	srv := &serverImpl{
		semaphore:        semaphore.NewWeighted(1),
		semaphoreTimeout: 5 * time.Millisecond,
	}

	// First should succeed
	err := srv.acquireSemaphore(ctx)
	require.NoError(t, err)

	// Second should time out
	err = srv.acquireSemaphore(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to acquire semaphore")

	// After releasing once, a new acquire should succeed
	srv.releaseSemaphore()
	err = srv.acquireSemaphore(ctx)
	require.NoError(t, err)
}
