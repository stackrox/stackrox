package or

import (
	"context"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/idcheck"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
)

type or struct {
	authorizers []authz.Authorizer
}

func (o or) Authorized(ctx context.Context) error {
	for _, a := range o.authorizers {
		if passed(a.Authorized(ctx)) {
			return nil
		}
	}
	return authz.ErrNotAuthorized{Explanation: "no authorizer could authorize this request"}
}

func passed(err error) bool {
	return err == nil
}

// Or creates an Authorizer that succeeds if any of the provided Authorizers succeed.
func Or(authorizers ...authz.Authorizer) authz.Authorizer {
	return or{
		authorizers: authorizers,
	}
}

// SensorOrUser returns an Authorizer that allows any authenticated user,
// or any sensor.
func SensorOrUser() authz.Authorizer {
	return Or(
		idcheck.SensorsOnly(),
		user.Any(),
	)
}
