package enrichment

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/imageintegration"
	imageintegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/pkg/images/enricher"
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
		enricher.New(imageintegration.Set()),
		risk.GetScorer()); err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Enricher to use.
func Singleton() Enricher {
	once.Do(initialize)
	return en
}
