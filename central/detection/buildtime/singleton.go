package buildtime

import (
	"context"

	"github.com/stackrox/stackrox/central/detection"
	policyDataStore "github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	policyUtils "github.com/stackrox/stackrox/pkg/policies"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
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
	policySet = detection.NewPolicySet(policyDataStore.Singleton())
	policies, err := policyDataStore.Singleton().GetAllPolicies(policyCtx)
	utils.CrashOnError(err)

	for _, policy := range policies {
		if policyUtils.AppliesAtBuildTime(policy) {
			utils.Must(policySet.UpsertPolicy(policy))
		}
	}

	detector = NewDetector(policySet)
}
