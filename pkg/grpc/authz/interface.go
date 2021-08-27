package authz

import (
	"context"
)

// An Authorizer tells if a context is properly authorized.
type Authorizer interface {
	// Authorized returns an error if authorization fails.
	// If authorization is successful, it returns nil.
	Authorized(ctx context.Context, fullMethodName string) error
}
