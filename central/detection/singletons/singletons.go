package singletons

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	enrichmentSingletons "bitbucket.org/stack-rox/apollo/central/enrichment/singletons"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	notifierProcessor "bitbucket.org/stack-rox/apollo/central/notifier/processor"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
)

var (
	once sync.Once

	detector *detection.Detector
)

func initialize() {
	var err error
	detector, err = detection.New(alertDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		policyDataStore.Singleton(),
		imageDataStore.Singleton(),
		enrichmentSingletons.GetEnricher(),
		notifierProcessor.Singleton())
	if err != nil {
		panic(err)
	}
}

// GetDetector provides the singleton detector to use.
func GetDetector() *detection.Detector {
	once.Do(initialize)
	return detector
}
