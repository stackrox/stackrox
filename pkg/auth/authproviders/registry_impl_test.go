package authproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	urlPathPrefix = "/sso/"
	redirectURL   = "/auth/response/generic"
)

func TestDeleteAuthProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	providerStore := mocks.NewMockStore(mockCtrl)

	registry := NewStoreBackedRegistry(
		urlPathPrefix,
		redirectURL,
		providerStore,
		&tstTokenIssuerFactory{},
		&tstRoleMapperFactory{},
	)

	err := registry.RegisterBackendFactory(ctx, oidc.TypeName, oidc.NewFactory)
	require.NoError(t, err)
	err = registry.RegisterBackendFactory(ctx, saml.TypeName, saml.NewFactory)
	require.NoError(t, err)

	providerTraits := &storage.Traits{
		MutabilityMode: storage.Traits_ALLOW_MUTATE,
	}

	oidcProvider := &storage.AuthProvider{
		Name:       "OIDC Provider",
		Type:       oidc.TypeName,
		UiEndpoint: "localhost:8000",
		Enabled:    true,
		Traits:     providerTraits,
	}

	samlProvider := &storage.AuthProvider{
		Name:       "SAML Provider",
		Type:       saml.TypeName,
		UiEndpoint: "localhost:8000",
		Enabled:    true,
		Traits:     providerTraits,
	}

	provider1, err := registry.CreateProvider(ctx, WithStorageView(oidcProvider))
	assert.NoError(t, err)
	assert.Equal(t, oidc.TypeName, provider1.StorageView().GetType())

	provider2, err := registry.CreateProvider(ctx, WithStorageView(samlProvider))
	assert.NoError(t, err)
	assert.Equal(t, saml.TypeName, provider2.StorageView().GetType())

	err = registry.DeleteProvider(ctx, provider1.ID(), true, true)
	assert.NoError(t, err)

	err = registry.DeleteProvider(ctx, provider2.ID(), true, true)
	assert.NoError(t, err)
}
