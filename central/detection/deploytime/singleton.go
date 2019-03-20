package deploytime

import (
	"github.com/stackrox/rox/pkg/sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/deployment"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
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
	policySet = deployment.NewPolicySet(policyDataStore.Singleton(), processDataStore.Singleton())
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

	detector = NewDetector(policySet, deploymentDataStore.Singleton())
}
