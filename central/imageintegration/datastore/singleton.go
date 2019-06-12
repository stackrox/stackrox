package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
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
	toNotifyOfIntegrations := imageintegration.ToNotify()
	integrations, err := ad.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		log.Errorf("unable to use previous integrations: %s", err)
	}
	for _, ii := range integrations {
		if err := toNotifyOfIntegrations.NotifyUpdated(ii); err != nil {
			log.Errorf("unable to use previous integration: %s", err)
		}
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
