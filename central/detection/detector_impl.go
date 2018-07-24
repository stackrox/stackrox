package detection

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	notifierProcessor "bitbucket.org/stack-rox/apollo/central/notifier/processor"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
)

// Detector processes deployments and reports alerts if policies are violated.
type detectorImpl struct {
	alertStorage      alertDataStore.DataStore
	deploymentStorage deploymentDataStore.DataStore
	policyStorage     policyDataStore.DataStore
	imageStorage      imageDataStore.DataStore

	enricher              enrichment.Enricher
	notificationProcessor notifierProcessor.Processor
	taskC                 chan Task
	stoppedC              chan struct{}

	policyMutex sync.RWMutex
	policies    map[string]*matcher.Policy
}

// Stop closes the Task reprocessing channel, and waits for remaining tasks to finish before returning.
func (d *detectorImpl) Stop() {
	close(d.taskC)
	<-d.stoppedC
}
