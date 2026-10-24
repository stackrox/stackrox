package d

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// customAuth is an authorizer that does NOT come from an allowed constructor.
type customAuth struct{}

func (customAuth) Authorized(_ context.Context, _ string) error { return nil }

var badAuth = customAuth{}

// This violates the UserAuthorizerVerifier constraint. The package carries no diagnostic
// markers because it is used exclusively in exclusion-path tests.
type svc1 struct{}

func (s *svc1) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, badAuth.Authorized(ctx, fullMethodName)
}

// This is valid so the package is not trivially empty.
type svc2 struct{}

func (s *svc2) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, user.WithRole("Admin").Authorized(ctx, fullMethodName)
}
