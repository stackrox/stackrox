package saml

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	mockPermissions "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	mockTokens "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:embed testdata/cert.pem
var cert string

const (
	urlPathPrefix = "/sso/"
	redirectURL   = "/auth/response/generic"
)

func TestDeleteAuthProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	providerStore := mocks.NewMockStore(mockCtrl)
	issuerFactory := mockTokens.NewMockIssuerFactory(mockCtrl)
	roleMapperFactory := mockPermissions.NewMockRoleMapperFactory(mockCtrl)

	issuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	roleMapper := mockPermissions.NewMockRoleMapper(mockCtrl)
	roleMapperFactory.EXPECT().GetRoleMapper(gomock.Any()).Times(1).Return(roleMapper)

	registry := authproviders.NewStoreBackedRegistry(
		urlPathPrefix,
		redirectURL,
		providerStore,
		issuerFactory,
		roleMapperFactory,
	)

	err := registry.RegisterBackendFactory(ctx, TypeName, NewFactory)
	require.NoError(t, err)

	providerTraits := &storage.Traits{
		MutabilityMode: storage.Traits_ALLOW_MUTATE,
	}

	testProvider := &storage.AuthProvider{
		Name: "SAML Provider",
		Type: TypeName,
		Config: map[string]string{
			"sp_issuer":         "test issuer",
			"idp_issuer":        "test IDP issuer",
			"idp_sso_url":       "test SSO URL",
			"idp_cert_pem":      cert,
			"idp_nameid_format": "test Name ID format",
		},
		UiEndpoint: "localhost:8000",
		Enabled:    true,
		Traits:     providerTraits,
	}

	providerStore.EXPECT().AddAuthProvider(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	providerStore.EXPECT().GetAuthProvider(gomock.Any(), gomock.Any()).Times(1).Return(testProvider, true, nil)

	provider, err := registry.CreateProvider(ctx, authproviders.WithStorageView(testProvider))
	assert.NoError(t, err)
	assert.Equal(t, TypeName, provider.StorageView().GetType())

	providerStore.EXPECT().RemoveAuthProvider(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
	issuerFactory.EXPECT().UnregisterSource(gomock.Any()).Times(1).Return(nil)

	err = registry.DeleteProvider(ctx, provider.ID(), true, true)
	assert.NoError(t, err)
}
