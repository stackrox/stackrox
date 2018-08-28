package authz

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// An Authorizer tells if a context is properly authorized.
type Authorizer interface {
	// Authorized returns an error if authorization fails.
	// If authorization is successful, it returns nil.
	Authorized(ctx context.Context, fullMethodName string) error
}

// ErrNoCredentials occurs if no relevant credentials can be found.
var ErrNoCredentials = status.Error(codes.Unauthenticated, "required credentials not found")

// ErrNotAuthorized occurs if credentials are found, but they are
// insufficiently authorized.
func ErrNotAuthorized(explanation string) error {
	return status.Errorf(codes.PermissionDenied, "not authorized: %s", explanation)
}

// ErrNoAuthzConfigured occurs if authorization is not implemented for a
// service. This is a programming error.
var ErrNoAuthzConfigured = status.Error(codes.Unimplemented, "service authorization is misconfigured")

// ErrAuthnConfigMissing occurs if user authentication configuration is
// missing from the context, indicating that someone is asking for user info
// but has not included the auth interceptor (or the interceptor is
// malfunctioning). This is a programming error.
var ErrAuthnConfigMissing = status.Error(codes.Unimplemented, "authentication configuration could not be located")
