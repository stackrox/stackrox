package deploytime

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	"github.com/stackrox/rox/central/detection/utils"
	"github.com/stackrox/rox/central/enrichment"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	policyUtils "github.com/stackrox/rox/pkg/policies"
)

var (
	once sync.Once

	policySet deployment.PolicySet
	detector  Detector
)

// SingletonDetector returns the singleton instance of a Detector.
func SingletonDetector() Detector {
	once.Do(initialize)
	return detector
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() deployment.PolicySet {
	once.Do(initialize)
	return policySet
}

func initialize() {
	policySet = deployment.NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetPolicies()
	if err != nil {
		panic(err)
	}
	for _, policy := range policies {
		if policyUtils.AppliesAtDeployTime(policy) {
			if err := policySet.UpsertPolicy(policy); err != nil {
				panic(err)
			}
		}
	}

	detector = NewDetector(policySet,
		utils.SingletonAlertManager(),
		enrichment.Singleton(),
		deploymentDataStore.Singleton(),
	)
}
