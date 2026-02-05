package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
	siStore  pgStore.Store
)

func upsertDefaultRedHatSignatureIntegration(siStore store.SignatureIntegrationStore, signatureIntegration *storage.SignatureIntegration) error {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	log.Debugf("Upserting default Red Hat signature integration %q (%s)", signatureIntegration.GetName(), signatureIntegration.GetId())
	return siStore.Upsert(ctx, signatureIntegration)
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		siStore = pgStore.New(globaldb.GetPostgres())

		// Upsert default integration and start updater
		err := upsertDefaultRedHatSignatureIntegration(siStore, signatures.DefaultRedHatSignatureIntegration)
		utils.Should(errors.Wrap(err, "upserting default Red Hat signature integration"))

		// Start updating Red Hat's signing key dynamically.
		// This must be done after the previous upsert of `signatures.DefaultRedHatSignatureIntegration`, so that the
		// updated key takes precedence and starts working immediately in cases where it's been rotated after the ACS release.
		newUpdater().Start()

		instance = New(siStore, policyDataStore.Singleton())
	})
	return instance
}
