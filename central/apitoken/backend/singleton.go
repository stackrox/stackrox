package backend

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/apitoken/datastore"
	"github.com/stackrox/stackrox/central/jwt"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/tokens"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

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
				sac.ResourceScopeKeys(resources.APIToken)))

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
