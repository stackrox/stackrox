package deploytime

import (
	"context"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	policySet detection.PolicySet
	detector  Detector
)

// SingletonDetector returns the singleton instance of a Detector.
func SingletonDetector() Detector {
	once.Do(initialize)
	return detector
}

// SingletonPolicySet returns the singleton instance of a PolicySet.
func SingletonPolicySet() detection.PolicySet {
	once.Do(initialize)
	return policySet
}

func initialize() {
	policySet = detection.NewPolicySet(policyDataStore.Singleton(), detection.NewPolicyCompiler(matcher.DeploymentBuilderSingleton()))
	policies, err := policyDataStore.Singleton().GetPolicies(context.TODO())
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
