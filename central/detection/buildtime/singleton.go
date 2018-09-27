package buildtime

import (
	"sync"

	"github.com/stackrox/rox/central/detection/image"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	once sync.Once

	policySet image.PolicySet
	detector  Detector
)

func initialize() {
	policySet = image.NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetPolicies()
	if err != nil {
		panic(err)
	}
	for _, policy := range policies {
		if policy.GetLifecycleStage() == v1.LifecycleStage_BUILD_TIME {
			if err := policySet.UpsertPolicy(policy); err != nil {
				panic(err)
			}
		}
	}

	detector = NewDetector(policySet)
}

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
