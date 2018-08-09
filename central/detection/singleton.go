package detection

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
)

var (
	once sync.Once

	detector Detector
)

func initialize() {
	var err error
	detector, err = New(alertDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		policyDataStore.Singleton(),
		imageDataStore.Singleton(),
		enrichment.Singleton(),
		notifierProcessor.Singleton())
	if err != nil {
		panic(err)
	}
}

// GetDetector provides the singleton detector to use.
func GetDetector() Detector {
	once.Do(initialize)
	return detector
}
