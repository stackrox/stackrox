package phonehome

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
)

func TestConfig_HashUserID(t *testing.T) {
	cfg := &Config{
		ClientID: "test-client",
	}
	h := cfg.HashUserID("test-user", "test-provider")
	assert.Equal(t, hash("test-client:test-provider:test-user"), h)

	cfg = nil
	h = cfg.HashUserID("test-user", "test-provider")
	assert.Equal(t, hash("unknown:test-provider:test-user"), h)
}

func TestConfig_HashUserAuthID(t *testing.T) {
	cfg := &Config{
		ClientID: "test-client",
	}
	h := cfg.HashUserAuthID(nil)
	assert.Equal(t, hash("test-client:unknown:unauthenticated"), h)

	ctrl := gomock.NewController(t)
	id := mocks.NewMockIdentity(ctrl)
	provider, _ := authproviders.NewProvider(
		authproviders.WithID("test-provider"),
		authproviders.WithName("test-provider-name"),
	)
	id.EXPECT().UID().Return("test-id").Times(1)
	id.EXPECT().ExternalAuthProvider().Return(provider).Times(1)
	h = cfg.HashUserAuthID(id)
	assert.Equal(t, hash("test-client:test-provider:test-id"), h)
}
