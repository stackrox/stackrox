package authz

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
)

// An Authorizer tells if a context is properly authorized.
type Authorizer interface {
	// Authorized returns an error if authorization fails.
	// If authorization is successful, it returns nil.
	Authorized(context.Context) error
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

// HTTPStatus implements the HTTPStatus interface
func (e ErrNoCredentials) HTTPStatus() int {
	return runtime.HTTPStatusFromCode(e.Status())
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

// HTTPStatus implements the HTTPStatus interface
func (e ErrNotAuthorized) HTTPStatus() int {
	return runtime.HTTPStatusFromCode(e.Status())
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

// HTTPStatus implements the HTTPStatus interface
func (e ErrNoAuthzConfigured) HTTPStatus() int {
	return runtime.HTTPStatusFromCode(e.Status())
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

// HTTPStatus implements the HTTPStatus interface
func (e ErrAuthnConfigMissing) HTTPStatus() int {
	return runtime.HTTPStatusFromCode(e.Status())
}
