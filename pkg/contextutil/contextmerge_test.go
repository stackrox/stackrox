package contextutil

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMergeContext(t *testing.T) {
	t.Run("text first context is canceled", func(tt *testing.T) {
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
	t.Run("text second context is canceled", func(tt *testing.T) {
		defer goleak.VerifyNone(tt)
		ctx1 := context.Background()
		ctx2, cancel := context.WithCancel(context.Background())
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
		case <-ctx1.Done():
			tt.Error("the first context should not be canceled")
			tt.FailNow()
		case <-time.After(100 * time.Millisecond):
		}
	})
}
