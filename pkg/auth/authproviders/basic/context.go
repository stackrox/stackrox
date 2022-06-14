package basic

import (
	"context"

	"github.com/stackrox/stackrox/pkg/grpc/authn/basic"
)

type basicAuthMgrContextKey struct{}

// ContextWithBasicAuthManager injects the given basic auth manager into the context.
func ContextWithBasicAuthManager(ctx context.Context, mgr *basic.Manager) context.Context {
	return context.WithValue(ctx, basicAuthMgrContextKey{}, mgr)
}

func basicAuthManagerFromContext(ctx context.Context) *basic.Manager {
	mgr, _ := ctx.Value(basicAuthMgrContextKey{}).(*basic.Manager)
	return mgr
}
