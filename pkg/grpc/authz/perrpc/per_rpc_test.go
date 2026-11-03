package perrpc

import (
	"context"
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

// recordingAuthorizer captures the last rpcName it received.
type recordingAuthorizer struct {
	lastRPCName string
	err         error
}

func (r *recordingAuthorizer) Authorized(_ context.Context, rpcName string) error {
	r.lastRPCName = rpcName
	return r.err
}

var (
	allowAll  = fixedAuthorizer{}
	rejectAll = fixedAuthorizer{err: errox.NotAuthorized}
)

const (
	methodA = "/svc.Service/MethodA"
	methodB = "/svc.Service/MethodB"
)

func TestPerRPC_Authorized(t *testing.T) {
	tests := map[string]struct {
		perRPC  *perRPC
		rpcName string
		wantErr error
	}{
		"known RPC, mapped authorizer allows": {
			perRPC: &perRPC{
				Default:     rejectAll,
				Authorizers: map[string]authz.Authorizer{methodA: allowAll},
			},
			rpcName: methodA,
		},
		"known RPC, mapped authorizer denies": {
			perRPC: &perRPC{
				Default:     allowAll,
				Authorizers: map[string]authz.Authorizer{methodA: rejectAll},
			},
			rpcName: methodA,
			wantErr: errox.NotAuthorized,
		},
		"unknown RPC, default allows": {
			perRPC: &perRPC{
				Default:     allowAll,
				Authorizers: map[string]authz.Authorizer{methodA: rejectAll},
			},
			rpcName: methodB,
		},
		"unknown RPC, default denies": {
			perRPC: &perRPC{
				Default:     rejectAll,
				Authorizers: map[string]authz.Authorizer{methodA: allowAll},
			},
			rpcName: methodB,
			wantErr: errox.NotAuthorized,
		},
		"empty authorizers map falls through to default": {
			perRPC: &perRPC{
				Default:     allowAll,
				Authorizers: map[string]authz.Authorizer{},
			},
			rpcName: methodA,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.ErrorIs(it, tt.perRPC.Authorized(it.Context(), tt.rpcName), tt.wantErr)
		})
	}
}

func TestPerRPC_Authorized_ForwardsRPCName(t *testing.T) {
	t.Run("to matched authorizer", func(it *testing.T) {
		rec := &recordingAuthorizer{}
		pr := &perRPC{
			Default:     rejectAll,
			Authorizers: map[string]authz.Authorizer{methodA: rec},
		}
		err := pr.Authorized(it.Context(), methodA)
		assert.Equal(it, methodA, rec.lastRPCName)
		assert.NoError(it, err)
	})

	t.Run("to default authorizer", func(it *testing.T) {
		rec := &recordingAuthorizer{}
		pr := &perRPC{
			Default:     rec,
			Authorizers: map[string]authz.Authorizer{methodA: allowAll},
		}
		err := pr.Authorized(it.Context(), methodB)
		assert.Equal(it, methodB, rec.lastRPCName)
		assert.NoError(it, err)
	})
}

func TestFromMap(t *testing.T) {
	t.Run("empty map denies all methods", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{})
		require.NotNil(it, a)
		assert.ErrorIs(it, a.Authorized(it.Context(), methodA), errox.NoAuthzConfigured)
	})

	t.Run("mapped method uses its authorizer", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			allowAll: {methodA},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
	})

	t.Run("unmapped method uses deny default", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			allowAll: {methodA},
		})
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NoAuthzConfigured)
	})

	t.Run("multiple methods per authorizer all routed correctly", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			allowAll: {methodA, methodB},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
		assert.NoError(it, a.Authorized(it.Context(), methodB))
	})

	t.Run("multiple authorizers each method routed to its own", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			allowAll:  {methodA},
			rejectAll: {methodB},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NotAuthorized)
	})

	t.Run("duplicate rpcName returns deny.Everyone", func(it *testing.T) {
		// Two distinct authorizers mapping the same method — programming error.
		// FromMap must return deny.Everyone() directly; neither configured authorizer should be called.
		rec1 := &recordingAuthorizer{}
		rec2 := &recordingAuthorizer{}
		a := FromMap(map[authz.Authorizer][]string{
			rec1: {methodA},
			rec2: {methodA},
		})

		// deny.Everyone() always returns errox.NoAuthzConfigured regardless of method.
		assert.ErrorIs(it, a.Authorized(it.Context(), methodA), errox.NoAuthzConfigured)
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NoAuthzConfigured)

		// Neither configured authorizer should have been called.
		assert.Empty(it, rec1.lastRPCName, "first authorizer must not be called")
		assert.Empty(it, rec2.lastRPCName, "second authorizer must not be called")
	})
}
