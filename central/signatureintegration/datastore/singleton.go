package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/pkg/errox"
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

// upsertDefaultRedHatSignatureIntegration creates the default Red Hat signature integration if it doesn't exist.
func upsertDefaultRedHatSignatureIntegration(ctx context.Context, siStore store.SignatureIntegrationStore) error {
	log.Info("Upserting default Red Hat signature integration")
	err := siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
	return err
}

// deleteDefaultRedHatSignatureIntegration removes the default Red Hat signature integration if it exists.
func deleteDefaultRedHatSignatureIntegration(ctx context.Context, siStore store.SignatureIntegrationStore) error {
	log.Info("Deleting default Red Hat signature integration")
	err := siStore.Delete(ctx, signatures.DefaultRedHatSignatureIntegration.GetId())
	if errors.Is(err, errox.NotFound) {
		log.Debug("Default Red Hat signature integration did not exist")
		return nil
	}
	return err
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
		err := upsertDefaultRedHatSignatureIntegration(ctx, siStore)
		utils.Should(errors.Wrap(err, "upserting default Red Hat signature integration"))
	} else {
		err := deleteDefaultRedHatSignatureIntegration(ctx, siStore)
		utils.Should(errors.Wrap(err, "deleting default RedHat signature integration"))
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
