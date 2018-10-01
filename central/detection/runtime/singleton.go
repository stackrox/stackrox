package runtime

import (
	"sync"

	"github.com/stackrox/rox/central/detection/deployment"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	once sync.Once

	policySet deployment.PolicySet
)

func initialize() {
	policySet = deployment.NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetPolicies()
	if err != nil {
		panic(err)
	}
	for _, policy := range policies {
		if policy.GetLifecycleStage() == v1.LifecycleStage_RUN_TIME {
			if err := policySet.UpsertPolicy(policy); err != nil {
				panic(err)
			}
		}
	}
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() deployment.PolicySet {
	once.Do(initialize)
	return policySet
}
