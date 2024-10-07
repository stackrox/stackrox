package oidc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

const (
	urlPathPrefix = "/sso/"
	redirectURL   = "/auth/response/generic"
)

var (
	mockServerURL = ""
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

	server := httptest.NewTLSServer(http.HandlerFunc(fakeServerHandler))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	serverURL.Scheme += "+insecure"
	mockServerURL = server.URL

	testProvider := &storage.AuthProvider{
		Name: "OIDC Provider",
		Type: TypeName,
		Config: map[string]string{
			"client_id":                "test client",
			"do_not_use_client_secret": "true",
			"issuer":                   serverURL.String(),
			"mode":                     "auto",
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

func fakeServerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	if r.URL.Path == "/.well-known/openid-configuration" {
		rspFormatString := `{
			"issuer": %q,
			"response_types_supported": ["id_token"]
		}`
		rspString := fmt.Sprintf(rspFormatString, mockServerURL)
		_, _ = w.Write([]byte(rspString))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
