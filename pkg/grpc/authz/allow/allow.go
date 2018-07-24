package allow

import (
	"context"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
)

// Anonymous returns an Authorizer that allows all access, even if the client
// is not authenticated in any way.
//
// Use sparingly!
func Anonymous() authz.Authorizer {
	return anonymous{}
}

type anonymous struct{}

// Authorized allows all access, even if the client is not authenticated in any way.
func (anonymous) Authorized(context.Context, string) error {
	return nil
}
