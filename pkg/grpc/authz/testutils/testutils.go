// Package testutils provides authorizer implementations for testing gRPC
// authorization code.
package testutils

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// fixedAuthorizer always returns the same preset error (nil = allow).
type fixedAuthorizer struct {
	Err error
}

var _ authz.Authorizer = fixedAuthorizer{}

// FixedAuthorizer returns an Authorizer that always returns the given error
// (nil = allow). The testing.TB parameter is ignored.
func FixedAuthorizer(_ testing.TB, fixedError error) authz.Authorizer {
	return fixedAuthorizer{Err: fixedError}
}

// Authorized returns the preset error, or nil if none was set.
func (f fixedAuthorizer) Authorized(context.Context, string) error {
	return f.Err
}

// RecordingAuthorizer captures every Authorized call for inspection.
type RecordingAuthorizer struct {
	Calls    int
	Contexts []context.Context
	Methods  []string
	Err      error
}

var _ authz.Authorizer = (*RecordingAuthorizer)(nil)

// Authorized records the call and returns the preset error, or nil if none was set.
func (r *RecordingAuthorizer) Authorized(ctx context.Context, method string) error {
	r.Calls++
	r.Contexts = append(r.Contexts, ctx)
	r.Methods = append(r.Methods, method)
	return r.Err
}

// Allow returns an Authorizer that authorizes every request.
// The testing.TB parameter is ignored.
func Allow(_ testing.TB) authz.Authorizer {
	return fixedAuthorizer{}
}

// Deny returns an Authorizer that denies every request.
// The testing.TB parameter is ignored.
func Deny(_ testing.TB) authz.Authorizer {
	return fixedAuthorizer{Err: errox.NotAuthorized}
}
