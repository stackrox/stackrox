package and

import (
	"context"
	"errors"
	"fmt"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

type and struct {
	authorizers []authz.Authorizer
}

func (a *and) Authorized(ctx context.Context, fullMethodName string) error {
	var errs []error
	for _, a := range a.authorizers {
		if err := a.Authorized(ctx, fullMethodName); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return errox.NotAuthorized.CausedBy(fmt.Errorf("some authorizer could not authorize this request: %w",
			errors.Join(errs...)))
	}
	return nil
}

// And creates an Authorizer that succeeds if all of the provided Authorizers succeed.
func And(authorizers ...authz.Authorizer) authz.Authorizer {
	return &and{
		authorizers: authorizers,
	}
}
