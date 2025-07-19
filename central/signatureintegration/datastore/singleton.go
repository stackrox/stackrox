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
		log.Debug("Upserting default Red Hat signature integration")
		err := siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
		utils.Should(errors.Wrap(err, "upserting default Red Hat signature integration"))
	} else {
		log.Debug("Deleting default Red Hat signature integration")
		err := siStore.Delete(ctx, signatures.DefaultRedHatSignatureIntegration.GetId())
		if errors.Is(err, errox.NotFound) {
			log.Debug("Default Red Hat signature integration did not exist")
		} else {
			utils.Should(errors.Wrap(err, "deleting default RedHat signature integration"))
		}
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
