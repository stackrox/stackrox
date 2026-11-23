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

// Authorized returns the preset error, or nil if none was set.
func (f fixedAuthorizer) Authorized(context.Context, string) error {
	return f.Err
}

// FixedAuthorizer returns an Authorizer that always returns the given error
// (nil = allow). The testing.TB parameter is ignored.
func FixedAuthorizer(_ testing.TB, fixedError error) authz.Authorizer {
	return fixedAuthorizer{Err: fixedError}
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

// RecordingAuthorizer is an Authorizer that captures every Authorized call
// for inspection.
type RecordingAuthorizer interface {
	authz.Authorizer

	// CallCount returns the number of recorded Authorized calls.
	CallCount() int
	// CallContexts returns the contexts received by the recorded Authorized calls.
	CallContexts() []context.Context
	// CallMethods returns the method names received by the recorded Authorized calls.
	CallMethods() []string
}

// recordingAuthorizer is the unexported implementation of RecordingAuthorizer.
type recordingAuthorizer struct {
	calls    int
	contexts []context.Context
	methods  []string
	err      error
}

var _ RecordingAuthorizer = (*recordingAuthorizer)(nil)

// Authorized records the call and returns the preset error, or nil if none was set.
func (r *recordingAuthorizer) Authorized(ctx context.Context, method string) error {
	r.calls++
	r.contexts = append(r.contexts, ctx)
	r.methods = append(r.methods, method)
	return r.err
}

// CallCount returns the number of recorded Authorized calls.
func (r *recordingAuthorizer) CallCount() int {
	return r.calls
}

// CallContexts returns the contexts received by the recorded Authorized calls.
func (r *recordingAuthorizer) CallContexts() []context.Context {
	return r.contexts
}

// CallMethods returns the method names received by the recorded Authorized calls.
func (r *recordingAuthorizer) CallMethods() []string {
	return r.methods
}

// NewRecordingAuthorizer returns a RecordingAuthorizer whose fields are all
// unset. The testing.TB parameter is ignored.
func NewRecordingAuthorizer(_ testing.TB) RecordingAuthorizer {
	return &recordingAuthorizer{}
}
