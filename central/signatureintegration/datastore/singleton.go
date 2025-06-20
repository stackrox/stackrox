package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

func initializeIntegrations(siStore store.SignatureIntegrationStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	var sis []*storage.SignatureIntegration
	err := siStore.Walk(ctx, func(si *storage.SignatureIntegration) error {
		sis = append(sis, si)
		return nil
	})
	utils.CrashOnError(err)

	// If we are starting from scratch, add the default signature integrations.
	if len(sis) == 0 {
		utils.Should(siStore.UpsertMany(ctx, store.DefaultSignatureIntegrations))
	}
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		if features.RedHatImagesSignedPolicy.Enabled() {
			initializeIntegrations(storage)
		}
		instance = New(storage, policyDataStore.Singleton())
	})
	return instance
}
