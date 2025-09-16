package contextutil

import (
	"context"
)

// MergeContext returns a context that will be canceled if either ctx1 or ctx2 are canceled.
// The returned stop function needs to be called after the returned ctx is canceled or
// the function that uses it finishes, otherwise if ctx1 is the context that triggered the cancellation,
// we will leak the AfterFunc goroutine.
func MergeContext(ctx1, ctx2 context.Context) (ctx context.Context, stop func() bool) {
	ctx, cancel := context.WithCancel(ctx1)
	stop = context.AfterFunc(ctx2, func() {
		cancel()
	})
	return ctx, stop
}
