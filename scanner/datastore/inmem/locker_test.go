package inmem

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocker(t *testing.T) {
	l := NewLocker()
	defer func() {
		assert.NoError(t, l.Close(context.Background()))
	}()

	ctx := context.Background()

	// TryLock acquires uncontended key.
	lCtx1, done1 := l.TryLock(ctx, "bundle-a")
	require.NoError(t, lCtx1.Err())

	// Second TryLock on same key returns canceled context immediately.
	lCtx2, done2 := l.TryLock(ctx, "bundle-a")
	assert.Error(t, lCtx2.Err())
	done2()

	// Different key is independent.
	lCtxB, doneB := l.TryLock(ctx, "bundle-b")
	require.NoError(t, lCtxB.Err())
	doneB()

	// Lock blocks until key is released or context expires.
	shortCtx, shortCancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer shortCancel()
	lCtxTimeout, doneTimeout := l.Lock(shortCtx, "bundle-a")
	assert.ErrorIs(t, lCtxTimeout.Err(), context.DeadlineExceeded)
	doneTimeout()

	// Release bundle-a in background; Lock should succeed.
	go func() {
		time.Sleep(15 * time.Millisecond)
		done1()
	}()
	lCtx3, done3 := l.Lock(ctx, "bundle-a")
	require.NoError(t, lCtx3.Err())
	done3()
	// Calling done twice must be safe.
	done3()
}
