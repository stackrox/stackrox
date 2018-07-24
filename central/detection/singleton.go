package detection

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	notifierProcessor "bitbucket.org/stack-rox/apollo/central/notifier/processor"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
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
