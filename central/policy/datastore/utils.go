package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

func markPoliciesAsCustom(policies ...*storage.Policy) {
	for _, policy := range policies {
		policy.SetIsDefault(false)
		policy.SetMitreVectorsLocked(false)
		policy.SetCriteriaLocked(false)
	}
}
