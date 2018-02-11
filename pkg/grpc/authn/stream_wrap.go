package authn

import (
	"context"

	"google.golang.org/grpc"
)

// StreamWithContext allows you to change the Context returned to future handlers.
type StreamWithContext struct {
	grpc.ServerStream
	ContextOverride context.Context // The context to return, which can be overwritten (initially nil).
}

// Context returns the context stored in ContextWrap.
func (w *StreamWithContext) Context() context.Context {
	if w.ContextOverride == nil {
		return w.ServerStream.Context()
	}
	return w.ContextOverride
}
