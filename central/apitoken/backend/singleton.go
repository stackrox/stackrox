package backend

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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

		// Create and initialize source.
		src := newSource()
		err := src.initFromStore(ctx, datastore.Singleton())
		utils.Should(errors.Wrap(err, "could not initialize API tokens source"))

		// Create token issuer.
		issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(src, tokens.WithDefaultTTL(defaultTTL))
		utils.Should(errors.Wrap(err, "could not create token issuer"))

		// Create the final backend.
		backendInstance = newBackend(issuer, src, datastore.Singleton())
	})
	return backendInstance
}
