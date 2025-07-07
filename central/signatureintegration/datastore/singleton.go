package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

// createDefaultRedHatSignatureIntegration creates the default Red Hat signature integration if it doesn't exist.
func createDefaultRedHatSignatureIntegration(ctx context.Context, siStore store.SignatureIntegrationStore) {
	if _, exists, err := siStore.Get(ctx, signatures.DefaultRedHatSignatureIntegration.GetId()); err != nil {
		utils.Should(errors.Wrap(err, "unable to detect if default Red Hat signature integration exists"))
		return
	} else if exists {
		log.Debug("Default Red Hat signature integration already exists, not upserting")
		return
	}

	log.Info("Upserting default Red Hat signature integration")
	err := siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
	utils.Should(errors.Wrap(err, "unable to upsert default Red Hat signature integration"))
}

// removeDefaultRedHatSignatureIntegration removes the default Red Hat signature integration if it exists.
func removeDefaultRedHatSignatureIntegration(ctx context.Context, siStore store.SignatureIntegrationStore) {
	if _, exists, err := siStore.Get(ctx, signatures.DefaultRedHatSignatureIntegration.GetId()); err != nil {
		utils.Should(errors.Wrap(err, "unable to detect if default Red Hat signature integration exists"))
		return
	} else if !exists {
		log.Debug("Default Red Hat signature integration does not exist, not deleting")
		return
	}

	log.Info("Deleting default Red Hat signature integration")
	err := siStore.Delete(ctx, signatures.DefaultRedHatSignatureIntegration.GetId())
	utils.Should(errors.Wrap(err, "unable to delete default Red Hat signature integration"))
}

// setupDefaultRedHatSignatureIntegration ensures the presence of the default Red Hat signature integration is
// in line with the RedHatImagesSignedPolicy feature flag.
func setupDefaultRedHatSignatureIntegration(siStore store.SignatureIntegrationStore) {
	log.Debugf("Setting up default Red Hat signature integration %q (%s)",
		signatures.DefaultRedHatSignatureIntegration.GetName(),
		signatures.DefaultRedHatSignatureIntegration.GetId())

	// Decide what to do based on the feature flag
	log.Debugf("RedHatImagesSignedPolicy feature flag is %t", features.RedHatImagesSignedPolicy.Enabled())

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	if features.RedHatImagesSignedPolicy.Enabled() {
		createDefaultRedHatSignatureIntegration(ctx, siStore)
	} else {
		removeDefaultRedHatSignatureIntegration(ctx, siStore)
	}
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		setupDefaultRedHatSignatureIntegration(storage)
		instance = New(storage, policyDataStore.Singleton())
	})
	return instance
}
