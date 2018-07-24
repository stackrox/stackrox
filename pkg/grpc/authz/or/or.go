package or

import (
	"context"

	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/idcheck"
)

type or struct {
	authorizers []authz.Authorizer
}

func (o *or) Authorized(ctx context.Context, fullMethodName string) error {
	var errors []error
	for _, a := range o.authorizers {
		err := a.Authorized(ctx, fullMethodName)
		if err == nil {
			return nil
		}
		errors = append(errors, err)
	}
	return authz.ErrNotAuthorized{
		Explanation: errorhelpers.NewErrorListWithErrors("no authorizer could authorize this request:", errors).String(),
	}
}

// Or creates an Authorizer that succeeds if any of the provided Authorizers succeed.
func Or(authorizers ...authz.Authorizer) authz.Authorizer {
	return &or{
		authorizers: authorizers,
	}
}

// SensorOrAuthorizer returns an Authorizer that allows authorizes any
// sensor, or anything that the passed authorizer authorizes.
func SensorOrAuthorizer(authorizer authz.Authorizer) authz.Authorizer {
	return Or(
		idcheck.SensorsOnly(),
		authorizer,
	)
}
