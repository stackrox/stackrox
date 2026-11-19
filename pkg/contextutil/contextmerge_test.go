package contextutil

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stretchr/testify/assert"
)

func TestMergeContext(t *testing.T) {
	t.Run("First context is canceled", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1, cancel := context.WithCancel(context.Background())
		ctx2 := context.Background()
		ctx, stopper := MergeContext(ctx1, ctx2)
		defer func() {
			_ = stopper()
		}()
		cancel()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			tt.Error("timeout waiting for the context to be canceled")
			tt.FailNow()
		}
		select {
		case <-ctx2.Done():
			tt.Error("the second context should not be canceled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("Second context is canceled", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1 := context.Background()
		ctx2, cancel := context.WithCancel(context.Background())
		ctx, stopper := MergeContext(ctx1, ctx2)
		defer func() {
			// If ctx2 is canceled, we don't need to call this stopper, but
			// it is still good to do it to illustrate the correct use of MergeContext
			_ = stopper()
		}()
		cancel()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			tt.Error("timeout waiting for the context to be canceled")
			tt.FailNow()
		}
		select {
		case <-ctx1.Done():
			tt.Error("the first context should not be canceled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("Both contexts are canceled", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2, cancel2 := context.WithCancel(context.Background())
		ctx, stopper := MergeContext(ctx1, ctx2)
		defer func() {
			// If ctx2 is canceled, we don't need to call this stopper, but
			// it is still good to do it to illustrate the correct use of MergeContext
			_ = stopper()
		}()
		cancel1()
		cancel2()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			tt.Error("timeout waiting for the context to be canceled")
			tt.FailNow()
		}
	})
	t.Run("Second context is canceled without calling stop should not leak", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1 := context.Background()
		ctx2, cancel2 := context.WithCancel(context.Background())
		// For this test we do not call the stop function,
		// and we should not have any goroutine leaks.
		// NOTICE: This is not the correct way use the MergeContext function.
		ctx, _ := MergeContext(ctx1, ctx2)
		cancel2()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			tt.Error("timeout waiting for the context to be canceled")
			tt.FailNow()
		}
		select {
		case <-ctx1.Done():
			tt.Error("the first context should not be canceled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("call stop should not cancel ctx1, ctx2, nor the merged ctx", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1 := context.Background()
		ctx2 := context.Background()
		ctx, stopper := MergeContext(ctx1, ctx2)
		assert.True(tt, stopper())
		select {
		case <-ctx1.Done():
			tt.Error("ctx1 should not be cancelled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
		select {
		case <-ctx2.Done():
			tt.Error("ctx2 should not be cancelled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
		select {
		case <-ctx.Done():
			tt.Error("merged context should not be cancelled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("after stop, canceling ctx2 should not cancel merged ctx", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1 := context.Background()
		ctx2, cancel2 := context.WithCancel(context.Background())
		ctx, stopper := MergeContext(ctx1, ctx2)
		assert.True(tt, stopper())
		cancel2()
		select {
		case <-ctx.Done():
			tt.Error("merged context should not be canceled after stop")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
	t.Run("after stop, canceling ctx1 should still cancel merged ctx", func(tt *testing.T) {
		goleak.AssertNoGoroutineLeaks(tt)
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2 := context.Background()
		ctx, stopper := MergeContext(ctx1, ctx2)
		assert.True(tt, stopper())
		cancel1()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			tt.Error("merged context should be canceled after stop")
			tt.FailNow()
		}
	})
}
