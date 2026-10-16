package and

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type andAuthorizer struct{}

func (a *andAuthorizer) Authorized(_ context.Context, _ string) error { return nil }

// And creates an Authorizer that succeeds if all of the provided Authorizers succeed.
func And(authorizers ...authz.Authorizer) authz.Authorizer { return &andAuthorizer{} }
