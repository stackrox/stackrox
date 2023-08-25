package contextutil

import (
	"context"
	"time"
)

// ContextWithTimeoutIfNotExists returns a context with a timeout if one does not exist
func ContextWithTimeoutIfNotExists(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	_, ok := ctx.Deadline()
	if ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
