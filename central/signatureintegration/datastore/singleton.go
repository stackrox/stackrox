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

func createDefaultRedHatSignatureIntegration(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	if _, exists, err := siStore.Get(ctx, signatures.DefaultRedHatSignatureIntegration.GetId()); err != nil {
		utils.Should(errors.Wrap(err, "unable to detect if default Red Hat signature integration exists"))
		return
	} else if exists {
		// Nothing to do, integration exists.
		return
	}

	log.Infof("Upserting default Red Hat signature integration %q (%v)",
		signatures.DefaultRedHatSignatureIntegration.GetName(),
		signatures.DefaultRedHatSignatureIntegration.GetId())
	err := siStore.Upsert(ctx, signatures.DefaultRedHatSignatureIntegration)
	utils.Should(errors.Wrap(err, "unable to upsert default Red Hat signature integration"))
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		if features.RedHatImagesSignedPolicy.Enabled() {
			createDefaultRedHatSignatureIntegration(storage)
		}
		instance = New(storage, policyDataStore.Singleton())
	})
	return instance
}
