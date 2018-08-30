package deploytime

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	once sync.Once

	policySet    PolicySet
	alertManager AlertManager
	detector     Detector
)

func initialize() {
	policySet = NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetPolicies()
	if err != nil {
		panic(err)
	}
	for _, policy := range policies {
		if policy.GetLifecycleStage() == v1.LifecycleStage_DEPLOY_TIME {
			if err := policySet.UpsertPolicy(policy); err != nil {
				panic(err)
			}
		}
	}

	alertManager = NewAlertManager(notifierProcessor.Singleton(), alertDataStore.Singleton())

	detector = NewDetector(policySet,
		alertManager,
		enrichment.Singleton(),
		deploymentDataStore.Singleton())
}

// SingletonDetector returns the singleton instance of a Detector.
func SingletonDetector() Detector {
	once.Do(initialize)
	return detector
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() PolicySet {
	once.Do(initialize)
	return policySet
}
