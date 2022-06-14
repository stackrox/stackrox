package and

import (
	"context"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

type and struct {
	authorizers []authz.Authorizer
}

func (a *and) Authorized(ctx context.Context, fullMethodName string) error {
	var errors []error
	for _, a := range a.authorizers {
		if err := a.Authorized(ctx, fullMethodName); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		return errox.NotAuthorized.CausedBy(errorhelpers.NewErrorListWithErrors("some authorizer could not authorize this request:", errors).String())
	}
	return nil
}

// And creates an Authorizer that succeeds if all of the provided Authorizers succeed.
func And(authorizers ...authz.Authorizer) authz.Authorizer {
	return &and{
		authorizers: authorizers,
	}
}
