package backend

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	id           = `https://stackrox.io/jwt-sources#api-tokens`
	httpScopedID = `https://stackrox.io/jwt-sources#api-tokens-http-scoped`
	apiToken     = "api-token"
)

var (
	backendInstance     Backend
	initBackendInstance sync.Once
)

// Singleton returns the apitoken backend singleton instance.
func Singleton() Backend {
	initBackendInstance.Do(func() {
		// Enable access to tokens for initialization.
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Integration)))

		var err error
		backendInstance, err = initBackend(ctx, jwt.IssuerFactorySingleton(), datastore.Singleton())
		utils.Should(err)
	})
	return backendInstance
}

func initBackend(ctx context.Context, factory tokens.IssuerFactory, tokenStore datastore.DataStore) (Backend, error) {
	revocationLayer := tokens.NewRevocationLayer()
	provider := tokenbased.NewTokenAuthProvider(
		id,
		apiToken,
		apiToken,
		tokenbased.WithRevocationLayer(revocationLayer),
	)
	scopedProvider := tokenbased.NewTokenAuthProvider(
		httpScopedID,
		apiToken,
		apiToken,
		tokenbased.WithRevocationLayer(revocationLayer),
		tokenbased.WithValidationLayer(tokenbased.NewHTTPScopedValidationLayer(httpScopedID)),
	)
	if err := provider.InitFromStore(ctx, tokenStore); err != nil {
		return nil, errors.Wrap(err, "could not initialize API tokens source")
	}

	issuer, err := factory.CreateIssuer(provider, tokens.WithDefaultTTL(defaultTTL))
	if err != nil {
		return nil, errors.Wrap(err, "could not create token issuer")
	}
	scopedIssuer, err := factory.CreateIssuer(scopedProvider, tokens.WithDefaultTTL(defaultTTL))
	if err != nil {
		return nil, errors.Wrap(err, "could not create HTTP-scoped token issuer")
	}
	return New(issuer, scopedIssuer, revocationLayer, tokenStore), nil
}
