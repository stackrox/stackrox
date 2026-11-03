package and

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedAuthorizer always returns the same preset error (nil = allow).
type fixedAuthorizer struct {
	err error
}

func (f fixedAuthorizer) Authorized(_ context.Context, _ string) error { return f.err }

// recordingAuthorizer captures every Authorized call for inspection.
type recordingAuthorizer struct {
	calls []struct {
		ctx        context.Context
		methodName string
	}
	err error
}

func (r *recordingAuthorizer) Authorized(ctx context.Context, methodName string) error {
	r.calls = append(r.calls, struct {
		ctx        context.Context
		methodName string
	}{ctx, methodName})
	return r.err
}

var (
	allow = fixedAuthorizer{}
	deny  = fixedAuthorizer{err: errox.NotAuthorized}
)

func TestAnd_Authorized(t *testing.T) {
	customErr := errors.New("custom error")

	tests := map[string]struct {
		authorizers []authz.Authorizer
		wantErr     error
	}{
		"no authorizers": {
			authorizers: nil,
		},
		"single authorizer allows": {
			authorizers: []authz.Authorizer{allow},
		},
		"single authorizer denies": {
			authorizers: []authz.Authorizer{deny},
			wantErr:     errox.NotAuthorized,
		},
		"all authorizers allow": {
			authorizers: []authz.Authorizer{allow, allow, allow},
		},
		"all authorizers deny": {
			authorizers: []authz.Authorizer{deny, deny, deny},
			wantErr:     errox.NotAuthorized,
		},
		"first denies, second allows": {
			authorizers: []authz.Authorizer{deny, allow},
			wantErr:     errox.NotAuthorized,
		},
		"first allows, second denies": {
			authorizers: []authz.Authorizer{allow, deny},
			wantErr:     errox.NotAuthorized,
		},
		"non-NotAuthorized error is wrapped in NotAuthorized": {
			authorizers: []authz.Authorizer{fixedAuthorizer{err: customErr}},
			wantErr:     errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			err := And(tt.authorizers...).Authorized(it.Context(), "/svc/Method")
			assert.ErrorIs(it, err, tt.wantErr)
		})
	}
}

// TestAnd_NoShortCircuit verifies all authorizers are evaluated regardless of failures.
func TestAnd_NoShortCircuit(t *testing.T) {
	second := &recordingAuthorizer{}
	And(deny, second).Authorized(context.Background(), "")
	assert.Len(t, second.calls, 1, "second authorizer must be called even when first fails")
}

// TestAnd_ForwardsContextAndMethod verifies ctx and methodName reach each sub-authorizer.
func TestAnd_ForwardsContextAndMethod(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{ key string }{"k"}, "v")
	const method = "/some.Service/Method"

	rec := &recordingAuthorizer{}
	And(rec).Authorized(ctx, method)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, ctx, rec.calls[0].ctx)
	assert.Equal(t, method, rec.calls[0].methodName)
}

func TestAnd_ReturnsAuthorizer(t *testing.T) {
	assert.NotNil(t, And())
}
