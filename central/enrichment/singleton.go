package enrichment

import (
	"sync"

	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	imageintegrationDataStore "bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	multiplierStore "bitbucket.org/stack-rox/apollo/central/multiplier/store"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/pkg/imageenricher"
)

var (
	once sync.Once

	en Enricher
)

func initialize() {
	var err error
	if en, err = New(deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		imageintegrationDataStore.Singleton(),
		multiplierStore.Singleton(),
		imageenricher.New(),
		risk.GetScorer()); err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Enricher to use.
func Singleton() Enricher {
	once.Do(initialize)
	return en
}
