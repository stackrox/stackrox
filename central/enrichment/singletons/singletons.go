package singletons

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	imageintegrationDataStore "bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	multiplierStore "bitbucket.org/stack-rox/apollo/central/multiplier/store"
	risk "bitbucket.org/stack-rox/apollo/central/risk/singletons"
)

var (
	once sync.Once

	enricher *enrichment.Enricher
)

func initialize() {
	var err error
	if enricher, err = enrichment.New(deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		imageintegrationDataStore.Singleton(),
		multiplierStore.Singleton(),
		alertDataStore.Singleton(),
		risk.GetScorer()); err != nil {
		panic(err)
	}
}

// GetEnricher provides the singleton Enricher to use.
func GetEnricher() *enrichment.Enricher {
	once.Do(initialize)
	return enricher
}
