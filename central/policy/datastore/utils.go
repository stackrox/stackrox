package datastore

import (
	"github.com/stackrox/stackrox/generated/storage"
)

func markPoliciesAsCustom(policies ...*storage.Policy) {
	for _, policy := range policies {
		policy.IsDefault = false
		policy.MitreVectorsLocked = false
		policy.CriteriaLocked = false
	}
}
