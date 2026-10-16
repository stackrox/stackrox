package authz

import "context"

// Authorizer checks whether the caller is authorized to invoke an RPC.
type Authorizer interface {
	Authorized(ctx context.Context, fullMethodName string) error
}
