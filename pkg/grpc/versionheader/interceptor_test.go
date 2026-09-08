package versionheader

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type fakeIdentity struct {
	authn.Identity
}

func TestUnaryServerInterceptor_Authenticated(t *testing.T) {
	testutils.SetMainVersion(t, "4.8.0")
	ctx := authn.ContextWithIdentity(context.Background(), fakeIdentity{}, t)

	interceptor := UnaryServerInterceptor()

	var handlerCalled bool
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

func TestUnaryServerInterceptor_Anonymous(t *testing.T) {
	testutils.SetMainVersion(t, "4.8.0")
	ctx := context.Background()

	interceptor := UnaryServerInterceptor()

	var handlerCalled bool
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

func TestVersionHeaderKey(t *testing.T) {
	assert.Equal(t, "rh-central-version", clientconn.CentralVersionHeader)
}

func TestSetVersionHeader_UsesCurrentVersion(t *testing.T) {
	testutils.SetMainVersion(t, "5.1.0")
	assert.Equal(t, "5.1.0", version.GetMainVersion())
}
