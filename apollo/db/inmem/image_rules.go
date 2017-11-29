package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type imageRuleStore struct {
	imageRules      map[string]*v1.ImageRule
	imageRulesMutex sync.Mutex

	persistent db.Storage
}

func newImageRuleStore(persistent db.Storage) *imageRuleStore {
	return &imageRuleStore{
		imageRules: make(map[string]*v1.ImageRule),
		persistent: persistent,
	}
}

func (s *imageRuleStore) loadFromPersistent() error {
	s.imageRulesMutex.Lock()
	defer s.imageRulesMutex.Unlock()
	rules, err := s.persistent.GetImageRules(&v1.GetImageRulesRequest{})
	if err != nil {
		return err
	}
	for _, rule := range rules {
		s.imageRules[rule.Name] = rule
	}
	return nil
}

// GetImageRules returns all image rules
func (s *imageRuleStore) GetImageRules(request *v1.GetImageRulesRequest) ([]*v1.ImageRule, error) {
	s.imageRulesMutex.Lock()
	defer s.imageRulesMutex.Unlock()
	rules := make([]*v1.ImageRule, 0, len(s.imageRules))
	for _, v := range s.imageRules {
		rules = append(rules, v)
	}
	if request.Name != "" {
		val, ok := s.imageRules[request.Name]
		if ok {
			return []*v1.ImageRule{val}, nil
		}
		return rules, nil
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules, nil
}

func (s *imageRuleStore) upsertImageRule(rule *v1.ImageRule) {
	s.imageRulesMutex.Lock()
	defer s.imageRulesMutex.Unlock()
	s.imageRules[rule.Name] = rule
}

// AddImageRule adds the image rule to the database
func (s *imageRuleStore) AddImageRule(rule *v1.ImageRule) error {
	s.imageRulesMutex.Lock()
	if _, ok := s.imageRules[rule.Name]; ok {
		return fmt.Errorf("rule with name %v already exists and cannot be added again", rule.Name)
	}
	s.imageRulesMutex.Unlock()
	if err := s.persistent.AddImageRule(rule); err != nil {
		return err
	}
	s.upsertImageRule(rule)
	return nil
}

// UpdateImageRule replaces the image rule stored with the new one
func (s *imageRuleStore) UpdateImageRule(rule *v1.ImageRule) error {
	if err := s.persistent.UpdateImageRule(rule); err != nil {
		return err
	}
	s.upsertImageRule(rule)
	return nil
}

// RemoveImageRule removes the image rule
func (s *imageRuleStore) RemoveImageRule(name string) error {
	s.imageRulesMutex.Lock()
	defer s.imageRulesMutex.Unlock()
	if err := s.persistent.RemoveImageRule(name); err != nil {
		return err
	}
	delete(s.imageRules, name)
	return nil
}
