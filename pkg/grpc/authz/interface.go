package authz

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
)

// An Authorizer tells if a context is properly authorized.
type Authorizer interface {
	// Authorized returns an error if authorization fails.
	// If authorization is successful, it returns nil.
	Authorized(ctx context.Context, fullMethodName string) error
}

// ErrNoCredentials occurs if no relevant credentials can be found.
type ErrNoCredentials struct{}

func (e ErrNoCredentials) Error() string {
	return "required credentials not found"
}

// Status implements the StatusError interface.
func (e ErrNoCredentials) Status() codes.Code {
	return codes.Unauthenticated
}

// ErrNotAuthorized occurs if credentials are found, but they are
// insufficiently authorized.
type ErrNotAuthorized struct {
	Explanation string
}

func (e ErrNotAuthorized) Error() string {
	return fmt.Sprintf("not authorized: %s", e.Explanation)
}

// Status implements the StatusError interface.
func (e ErrNotAuthorized) Status() codes.Code {
	return codes.PermissionDenied
}

// ErrNoAuthzConfigured occurs if authorization is not implemented for a
// service. This is a programming error.
type ErrNoAuthzConfigured struct{}

func (e ErrNoAuthzConfigured) Error() string {
	return "service authorization is misconfigured"
}

// Status implements the StatusError interface.
func (e ErrNoAuthzConfigured) Status() codes.Code {
	return codes.Unimplemented
}

// ErrAuthnConfigMissing occurs if user authentication configuration is
// missing from the context, indicating that someone is asking for user info
// but has not included the auth interceptor (or the interceptor is
// malfunctioning). This is a programming error.
type ErrAuthnConfigMissing struct{}

func (e ErrAuthnConfigMissing) Error() string {
	return "authentication configuration could not be located"
}

// Status implements the StatusError interface.
func (e ErrAuthnConfigMissing) Status() codes.Code {
	return codes.Unimplemented
}
