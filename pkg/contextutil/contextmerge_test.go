package contextutil

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMergeContext(t *testing.T) {
	t.Run("First context is canceled", func(tt *testing.T) {
		defer goleak.VerifyNone(tt)
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
		defer goleak.VerifyNone(tt)
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
		defer goleak.VerifyNone(tt)
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
		defer goleak.VerifyNone(tt)
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
}
