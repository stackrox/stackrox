package b

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type myAuth struct{}

func (m *myAuth) Authorized(_ context.Context, _ string) error { return nil }

// pkgAuth is a package-level variable: accepted by the package-level-only verifier.
var pkgAuth authz.Authorizer = &myAuth{}

// ---- Correct: package-level authorizer ----

type svc1 struct{}

func (s *svc1) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, pkgAuth.Authorized(ctx, fullMethodName)
}

// ---- Fail: struct field authorizer (not a package-level variable) ----

type svc2 struct {
	auth authz.Authorizer
}

func (s *svc2) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, s.auth.Authorized(ctx, fullMethodName)
}
