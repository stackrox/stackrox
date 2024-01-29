package observe

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

type collectAuthzTraceKey struct{}

// ContextWithAuthzTrace returns a context which is a child of the given context
// and contains the given instance of the authz trace.
func ContextWithAuthzTrace(ctx context.Context, trace *AuthzTrace) context.Context {
	return context.WithValue(ctx, collectAuthzTraceKey{}, trace)
}

// AuthzTraceFromContext returns mutable instance of authzTrace if present.
func AuthzTraceFromContext(ctx context.Context) *AuthzTrace {
	value := ctx.Value(collectAuthzTraceKey{})
	if value == nil {
		return nil
	}

	if authzTraceValue, ok := value.(*AuthzTrace); ok {
		return authzTraceValue
	}

	utils.Should(errors.Errorf("Per-request authorization trace is of type %T, expected %T", value, &AuthzTrace{}))
	return nil
}
