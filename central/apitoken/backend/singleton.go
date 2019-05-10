package backend

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	backendInstance     Backend
	initBackendInstance sync.Once
)

// Singleton returns the apitoken backend singleton instance.
func Singleton() Backend {
	initBackendInstance.Do(func() {
		// Create and initialize source.
		src := newSource()
		if err := src.initFromStore(context.TODO(), datastore.Singleton()); err != nil {
			log.Panicf("Could not initialize API tokens source: %v", err)
		}

		// Create token issuer.
		issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(src, tokens.WithDefaultTTL(defaultTTL))
		if err != nil {
			log.Panicf("Could not create token issuer: %v", err)
		}

		// Create the final backend.
		backendInstance = newBackend(issuer, src, datastore.Singleton())
	})
	return backendInstance
}
