package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
)

type policyStore struct {
	policies      map[string]*v1.Policy
	policiesMutex sync.Mutex

	persistent db.PolicyStorage
}

func newPolicyStore(persistent db.PolicyStorage) *policyStore {
	return &policyStore{
		policies:   make(map[string]*v1.Policy),
		persistent: persistent,
	}
}

func (s *policyStore) clone(policy *v1.Policy) *v1.Policy {
	return proto.Clone(policy).(*v1.Policy)
}

func (s *policyStore) loadFromPersistent() error {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()
	policies, err := s.persistent.GetPolicies(&v1.GetPoliciesRequest{})
	if err != nil {
		return err
	}
	for _, p := range policies {
		s.policies[p.Name] = p
	}
	return nil
}

// GetPolicy returns a policy by name.
func (s *policyStore) GetPolicy(name string) (p *v1.Policy, exist bool, err error) {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()
	p, exist = s.policies[name]
	return
}

// GetPolicies returns policies according to request.
func (s *policyStore) GetPolicies(request *v1.GetPoliciesRequest) (policies []*v1.Policy, err error) {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()

	namesSet := stringWrap(request.GetName()).asSet()
	categoriesSet := categoriesWrap(request.GetCategory()).asSet()
	severitiesSet := severitiesWrap(request.GetSeverity()).asSet()

	for _, p := range s.policies {
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

		policies = append(policies, s.clone(p))
	}

	sort.SliceStable(policies, func(i, j int) bool { return policies[i].Name < policies[j].Name })
	return
}

func (s *policyStore) matchCategories(alertCategories []v1.Policy_Category, categorySet map[v1.Policy_Category]struct{}) bool {
	for _, c := range alertCategories {
		if _, ok := categorySet[c]; ok {
			return true
		}
	}

	return false
}

// AddPolicy adds the policy to the database.
func (s *policyStore) AddPolicy(policy *v1.Policy) error {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()
	if _, ok := s.policies[policy.Name]; ok {
		return fmt.Errorf("policy with name %v already exists and cannot be added again", policy.Name)
	}
	if err := s.persistent.AddPolicy(policy); err != nil {
		return err
	}
	s.upsertPolicy(policy)
	return nil
}

// UpdatePolicy updates the policy.
func (s *policyStore) UpdatePolicy(policy *v1.Policy) error {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()
	if err := s.persistent.UpdatePolicy(policy); err != nil {
		return err
	}
	s.upsertPolicy(policy)
	return nil
}

func (s *policyStore) upsertPolicy(policy *v1.Policy) {
	s.policies[policy.Name] = s.clone(policy)
}

// RemovePolicy removes the policy.
func (s *policyStore) RemovePolicy(name string) error {
	s.policiesMutex.Lock()
	defer s.policiesMutex.Unlock()
	if err := s.persistent.RemovePolicy(name); err != nil {
		return err
	}
	delete(s.policies, name)
	return nil
}
