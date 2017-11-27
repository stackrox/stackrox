package inmem

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

func (i *InMemoryStore) loadImageRules() error {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	rules, err := i.persistent.GetImageRules(&v1.GetImageRulesRequest{})
	if err != nil {
		return err
	}
	for _, rule := range rules {
		i.imageRules[rule.Name] = rule
	}
	return nil
}

// GetImageRules returns all image rules
func (i *InMemoryStore) GetImageRules(request *v1.GetImageRulesRequest) ([]*v1.ImageRule, error) {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	rules := make([]*v1.ImageRule, 0, len(i.imageRules))
	for _, v := range i.imageRules {
		rules = append(rules, v)
	}
	if request.Name != "" {
		val, ok := i.imageRules[request.Name]
		if ok {
			return []*v1.ImageRule{val}, nil
		}
		return rules, nil
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules, nil
}

func (i *InMemoryStore) upsertImageRule(rule *v1.ImageRule) {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	i.imageRules[rule.Name] = rule
}

// AddImageRule adds the image rule to the database
func (i *InMemoryStore) AddImageRule(rule *v1.ImageRule) error {
	i.imageRulesMutex.Lock()
	if _, ok := i.imageRules[rule.Name]; ok {
		return fmt.Errorf("rule with name %v already exists and cannot be added again", rule.Name)
	}
	i.imageRulesMutex.Unlock()
	if err := i.persistent.AddImageRule(rule); err != nil {
		return err
	}
	i.upsertImageRule(rule)
	return nil
}

// UpdateImageRule replaces the image rule stored with the new one
func (i *InMemoryStore) UpdateImageRule(rule *v1.ImageRule) error {
	if err := i.persistent.UpdateImageRule(rule); err != nil {
		return err
	}
	i.upsertImageRule(rule)
	return nil
}

// RemoveImageRule removes the image rule
func (i *InMemoryStore) RemoveImageRule(name string) error {
	i.imageRulesMutex.Lock()
	defer i.imageRulesMutex.Unlock()
	if err := i.persistent.RemoveImageRule(name); err != nil {
		return err
	}
	delete(i.imageRules, name)
	return nil
}
