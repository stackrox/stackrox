package enrichanddetect

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
)

var (
	once sync.Once

	en   EnricherAndDetector
	loop Loop
)

func initialize() {
	en = New(enrichment.Singleton(), lifecycle.SingletonManager(), deploymentDataStore.Singleton(), imageDatastore.Singleton())
	loop = NewLoop(en, deploymentDataStore.Singleton())
}

// Singleton provides the singleton EnricherAndDetector to use.
func Singleton() EnricherAndDetector {
	once.Do(initialize)
	return en
}

// GetLoop provides the singleton Loop to use.
func GetLoop() Loop {
	once.Do(initialize)
	return loop
}
