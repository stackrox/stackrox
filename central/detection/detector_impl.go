package detection

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/pkg/compiledpolicies"
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
	policies    map[string]compiledpolicies.DeploymentMatcher
}

// Stop closes the Task reprocessing channel, and waits for remaining tasks to finish before returning.
func (d *detectorImpl) Stop() {
	close(d.taskC)
	<-d.stoppedC
}
