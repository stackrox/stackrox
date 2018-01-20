package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

type policyStore struct {
	db.PolicyStorage
}

func newPolicyStore(persistent db.PolicyStorage) *policyStore {
	return &policyStore{
		PolicyStorage: persistent,
	}
}

// GetPolicies returns policies according to request.
func (s *policyStore) GetPolicies(request *v1.GetPoliciesRequest) ([]*v1.Policy, error) {
	policies, err := s.PolicyStorage.GetPolicies(request)
	if err != nil {
		return nil, err
	}
	namesSet := stringWrap(request.GetName()).asSet()
	categoriesSet := categoriesWrap(request.GetCategory()).asSet()
	severitiesSet := severitiesWrap(request.GetSeverity()).asSet()

	filteredPolicies := policies[:0]
	for _, p := range policies {
		if len(request.GetDisabled()) == 1 && p.GetDisabled() != request.GetDisabled()[0] {
			continue
		}

		if _, ok := namesSet[p.GetName()]; len(namesSet) > 0 && !ok {
			continue
		}

		if _, ok := severitiesSet[p.GetSeverity()]; len(severitiesSet) > 0 && !ok {
			continue
		}

		if len(categoriesSet) > 0 && !s.matchCategories(p.GetCategories(), categoriesSet) {
			continue
		}

		filteredPolicies = append(filteredPolicies, p)
	}
	return filteredPolicies, nil
}

func (s *policyStore) matchCategories(alertCategories []v1.Policy_Category, categorySet map[v1.Policy_Category]struct{}) bool {
	for _, c := range alertCategories {
		if _, ok := categorySet[c]; ok {
			return true
		}
	}

	return false
}
