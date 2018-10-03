package buildtime

import (
	"sync"

	"github.com/stackrox/rox/central/detection/image"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	policyUtils "github.com/stackrox/rox/pkg/policies"
)

var (
	once sync.Once

	policySet image.PolicySet
	detector  Detector
)

// SingletonDetector returns the singleton instance of a Detector.
func SingletonDetector() Detector {
	once.Do(initialize)
	return detector
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() image.PolicySet {
	once.Do(initialize)
	return policySet
}

func initialize() {
	policySet = image.NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetPolicies()
	if err != nil {
		panic(err)
	}
	for _, policy := range policies {
		if policyUtils.AppliesAtBuildTime(policy) {
			if err := policySet.UpsertPolicy(policy); err != nil {
				panic(err)
			}
		}
	}

	detector = NewDetector(policySet)
}
