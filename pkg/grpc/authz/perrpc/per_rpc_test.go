package perrpc

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Default:     testutils.Deny(t),
				Authorizers: map[string]authz.Authorizer{methodA: testutils.Allow(t)},
			},
			rpcName: methodA,
		},
		"known RPC, mapped authorizer denies": {
			perRPC: &perRPC{
				Default:     testutils.Allow(t),
				Authorizers: map[string]authz.Authorizer{methodA: testutils.Deny(t)},
			},
			rpcName: methodA,
			wantErr: errox.NotAuthorized,
		},
		"unknown RPC, default allows": {
			perRPC: &perRPC{
				Default:     testutils.Allow(t),
				Authorizers: map[string]authz.Authorizer{methodA: testutils.Deny(t)},
			},
			rpcName: methodB,
		},
		"unknown RPC, default denies": {
			perRPC: &perRPC{
				Default:     testutils.Deny(t),
				Authorizers: map[string]authz.Authorizer{methodA: testutils.Allow(t)},
			},
			rpcName: methodB,
			wantErr: errox.NotAuthorized,
		},
		"empty authorizers map falls through to default": {
			perRPC: &perRPC{
				Default:     testutils.Allow(t),
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
		rec := &testutils.RecordingAuthorizer{}
		pr := &perRPC{
			Default:     testutils.Deny(it),
			Authorizers: map[string]authz.Authorizer{methodA: rec},
		}
		err := pr.Authorized(it.Context(), methodA)
		assert.Equal(it, []string{methodA}, rec.Methods)
		assert.NoError(it, err)
	})

	t.Run("to default authorizer", func(it *testing.T) {
		rec := &testutils.RecordingAuthorizer{}
		pr := &perRPC{
			Default:     rec,
			Authorizers: map[string]authz.Authorizer{methodA: testutils.Allow(it)},
		}
		err := pr.Authorized(it.Context(), methodB)
		assert.Equal(it, []string{methodB}, rec.Methods)
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
			testutils.Allow(it): {methodA},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
	})

	t.Run("unmapped method uses deny default", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			testutils.Allow(it): {methodA},
		})
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NoAuthzConfigured)
	})

	t.Run("multiple methods per authorizer all routed correctly", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			testutils.Allow(it): {methodA, methodB},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
		assert.NoError(it, a.Authorized(it.Context(), methodB))
	})

	t.Run("multiple authorizers each method routed to its own", func(it *testing.T) {
		a := FromMap(map[authz.Authorizer][]string{
			testutils.Allow(it): {methodA},
			testutils.Deny(it):  {methodB},
		})
		assert.NoError(it, a.Authorized(it.Context(), methodA))
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NotAuthorized)
	})

	t.Run("duplicate rpcName returns deny.Everyone", func(it *testing.T) {
		// Two distinct authorizers mapping the same method — programming error.
		// FromMap must return deny.Everyone() directly; neither configured authorizer should be called.
		rec1 := &testutils.RecordingAuthorizer{}
		rec2 := &testutils.RecordingAuthorizer{}
		a := FromMap(map[authz.Authorizer][]string{
			rec1: {methodA},
			rec2: {methodA},
		})

		// deny.Everyone() always returns errox.NoAuthzConfigured regardless of method.
		assert.ErrorIs(it, a.Authorized(it.Context(), methodA), errox.NoAuthzConfigured)
		assert.ErrorIs(it, a.Authorized(it.Context(), methodB), errox.NoAuthzConfigured)

		// Neither configured authorizer should have been called.
		assert.Zero(it, rec1.Calls, "first authorizer must not be called")
		assert.Zero(it, rec2.Calls, "second authorizer must not be called")
	})
}
