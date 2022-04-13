package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/enrichment"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/imageintegration/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	// Create underlying store and datastore.
	storage := store.New(globaldb.GetGlobalDB())
	ad = New(storage)

	// Initialize the integration set with all present integrations.
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	integrationManager := enrichment.ManagerSingleton()
	integrations, err := ad.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		log.Errorf("unable to use previous integrations: %s", err)
	}
	for _, ii := range integrations {
		if err := integrationManager.Upsert(ii); err != nil {
			log.Errorf("unable to use previous integration: %s", err)
		}
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
