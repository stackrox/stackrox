package deploytime

import (
	"context"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	detectionPkg "github.com/stackrox/rox/pkg/detection"
	policyUtils "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	policySet detection.PolicySet
	detector  Detector

	policyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
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
	policySet = detection.NewPolicySet(policyDataStore.Singleton(), detectionPkg.NewPolicyCompiler(searchbasedpolicies.DeploymentBuilderSingleton()))
	policies, err := policyDataStore.Singleton().GetAllPolicies(policyCtx)
	utils.Must(err)

	for _, policy := range policies {
		if policyUtils.AppliesAtDeployTime(policy) {
			utils.Must(policySet.UpsertPolicy(policy))
		}
	}

	detector = NewDetector(policySet, deploymentDataStore.Singleton())
}
