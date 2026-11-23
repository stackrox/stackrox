package and

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			authorizers: []authz.Authorizer{testutils.Allow(t)},
		},
		"single authorizer denies": {
			authorizers: []authz.Authorizer{testutils.Deny(t)},
			wantErr:     errox.NotAuthorized,
		},
		"all authorizers allow": {
			authorizers: []authz.Authorizer{testutils.Allow(t), testutils.Allow(t), testutils.Allow(t)},
		},
		"all authorizers deny": {
			authorizers: []authz.Authorizer{testutils.Deny(t), testutils.Deny(t), testutils.Deny(t)},
			wantErr:     errox.NotAuthorized,
		},
		"first denies, second allows": {
			authorizers: []authz.Authorizer{testutils.Deny(t), testutils.Allow(t)},
			wantErr:     errox.NotAuthorized,
		},
		"first allows, second denies": {
			authorizers: []authz.Authorizer{testutils.Allow(t), testutils.Deny(t)},
			wantErr:     errox.NotAuthorized,
		},
		"non-NotAuthorized error is wrapped in NotAuthorized": {
			authorizers: []authz.Authorizer{testutils.FixedAuthorizer(t, customErr)},
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
	second := &testutils.RecordingAuthorizer{}
	err := And(testutils.Deny(t), second).Authorized(context.Background(), "")
	assert.ErrorIs(t, err, errox.NotAuthorized)
	assert.Equal(t, 1, second.Calls, "second authorizer must be called even when first fails")
}

// TestAnd_ForwardsContextAndMethod verifies ctx and methodName reach each sub-authorizer.
func TestAnd_ForwardsContextAndMethod(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{ key string }{"k"}, "v")
	const method = "/some.Service/Method"

	rec := &testutils.RecordingAuthorizer{}
	err := And(rec).Authorized(ctx, method)

	assert.NoError(t, err)
	require.Len(t, rec.Contexts, 1)
	assert.Equal(t, ctx, rec.Contexts[0])
	assert.Equal(t, method, rec.Methods[0])
}

func TestAnd_ReturnsAuthorizer(t *testing.T) {
	assert.NotNil(t, And())
}
