package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// PolicyRuleSet representss a combined set of PolicyRules.
type PolicyRuleSet interface {
	Add(prs ...*storage.PolicyRule)
	Grants(prs ...*storage.PolicyRule) bool
	ToSlice() []*storage.PolicyRule
}

// NewPolicyRuleSet returns a new instance of a PolicyRuleSet.
func NewPolicyRuleSet(fields ...PolicyRuleField) PolicyRuleSet {
	return &policyRuleSet{
		fields: NewPolicyRuleFieldSet(fields...),
	}
}

type policyRuleSet struct {
	fields  PolicyRuleFieldSet
	granted []*storage.PolicyRule
}

// AddAll adds all of the inputs to the set.
func (p *policyRuleSet) Add(prs ...*storage.PolicyRule) {
	for _, pr := range prs {
		p.add(pr)
	}
}

// GrantsAll returns if the set of PolicyRules grants all necessary permissions for the given list of policy rules.
func (p *policyRuleSet) Grants(prs ...*storage.PolicyRule) bool {
	for _, pr := range prs {
		if !p.grants(pr) {
			return false
		}
	}
	return true
}

// ToSlice returns a sorted list of policy rules with the rules broken up by apiGroup and resource.
func (p *policyRuleSet) ToSlice() []*storage.PolicyRule {
	if len(p.granted) == 0 {
		return nil
	}
	return p.granted
}

func (p *policyRuleSet) add(pr *storage.PolicyRule) {
	if p.grants(pr) {
		return // already granted
	}
	if p.tryReplace(pr) {
		return // grants greater permissions than existing rule
	}
	if p.tryMerge(pr) {
		return // combined with existing rule to expand permissions
	}
	// Needs to be appended as a new rule.
	p.granted = append(p.granted, pr)
}

func (p *policyRuleSet) grants(pr *storage.PolicyRule) bool {
	for _, rule := range p.granted {
		if p.fields.Grants(rule, pr) {
			return true
		}
	}
	return false
}

func (p *policyRuleSet) tryReplace(pr *storage.PolicyRule) bool {
	for index, rule := range p.granted {
		if p.fields.Grants(pr, rule) {
			p.granted[index] = pr
			return true
		}
	}
	return false
}

func (p *policyRuleSet) tryMerge(pr *storage.PolicyRule) bool {
	for _, rule := range p.granted {
		if p.fields.Merge(rule, pr) {
			return true
		}
	}
	return false
}
