package auth

import (
	"context"

	"google.golang.org/grpc"
)

// streamWithContext allows you to change the Context returned to future handlers.
type streamWithContext struct {
	grpc.ServerStream
	ContextOverride context.Context // The context to return, which can be overwritten (initially nil).
}

// Context returns the context stored in ContextWrap.
func (w *streamWithContext) Context() context.Context {
	if w.ContextOverride == nil {
		return w.ServerStream.Context()
	}
	return w.ContextOverride
}
