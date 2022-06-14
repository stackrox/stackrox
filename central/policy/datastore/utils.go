package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

func markPoliciesAsCustom(policies ...*storage.Policy) {
	for _, policy := range policies {
		policy.IsDefault = false
		policy.MitreVectorsLocked = false
		policy.CriteriaLocked = false
	}
}
