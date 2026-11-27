package interceptor

import (
	"context"
)

// registeringHandlerBase wraps a handler that records whether it was called.
type registeringHandlerBase struct {
	called bool
}

func (b *registeringHandlerBase) Handler(_ context.Context, _ interface{}) (interface{}, error) {
	b.called = true
	return nil, nil
}

// serverWithAuthorizer implements AuthFuncOverride with a preset error (nil = allow).
type serverWithAuthorizer struct {
	authZError error
}

func (s *serverWithAuthorizer) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	return ctx, s.authZError
}

// serverWithoutAuthFuncOverride implements no AuthFuncOverride.
type serverWithoutAuthFuncOverride struct{}
