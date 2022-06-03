package datastore

import (
	"context"

	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/imageintegration/store/bolt"
	"github.com/stackrox/rox/central/imageintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initializeDefaultIntegrations(ctx context.Context, storage store.Store) {
	integrations, err := ad.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	utils.CrashOnError(err)
	if !env.OfflineModeEnv.BooleanSetting() && len(integrations) == 0 {
		// Add default integrations
		for _, ii := range store.DefaultImageIntegrations {
			utils.Must(storage.Upsert(ctx, ii))
		}
	}
}

func initializeManager(ctx context.Context) {
	// Initialize the integration set with all present integrations.
	integrationManager := enrichment.ManagerSingleton()
	integrations, err := ad.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		log.Errorf("unable to use previous integrations: %s", err)
	}
	for _, ii := range integrations {
		if err := integrationManager.Upsert(ii); err != nil {
			log.Errorf("unable to use previous integration %s: %v", ii.GetName(), err)
		}
	}
}

func initialize() {
	// Create underlying store and datastore.
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	ad = New(storage)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	initializeDefaultIntegrations(ctx, storage)
	initializeManager(ctx)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
