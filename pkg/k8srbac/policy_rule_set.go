package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// PolicyRuleSet represents a combined set of PolicyRules.
type PolicyRuleSet interface {
	Add(prs ...*storage.PolicyRule)
	Grants(prs ...*storage.PolicyRule) bool
	GetPermissionMap() map[string]set.StringSet
	VerbSet() set.StringSet
	ResourceSet() set.StringSet
	NonResourceURLSet() set.StringSet
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

// Add adds all the policy rules to the set.
func (p *policyRuleSet) Add(prs ...*storage.PolicyRule) {
	for _, pr := range prs {
		// policyRuleSet can *take ownership of and later modify* its
		// parameter on subsequent calls. It is better to play it safe and Clone the rules
		// before adding.
		p.add(pr.Clone())
	}
}

// Grants returns if the set of PolicyRules grants all necessary permissions for the given list of policy rules.
func (p *policyRuleSet) Grants(prs ...*storage.PolicyRule) bool {
	for _, pr := range prs {
		if !p.grants(pr) {
			return false
		}
	}
	return true
}

// VerbSet returns if the set of verbs granted by the given list of policy rules
func (p *policyRuleSet) VerbSet() set.StringSet {
	verbs := set.NewStringSet()
	for _, rule := range p.granted {
		for _, verb := range rule.GetVerbs() {
			verbs.Add(verb)
		}
	}
	return verbs
}

// ResourceSet returns the set of resources that permissions have been  granted to by the given list of policy rules
func (p *policyRuleSet) ResourceSet() set.StringSet {
	resources := set.NewStringSet()
	for _, rule := range p.granted {
		for _, resource := range rule.GetResources() {
			resources.Add(resource)
		}
	}
	return resources
}

// NonResourceURLSet returns the set of non resource URLs that permissions have been granted to by the given list of policy rules
func (p *policyRuleSet) NonResourceURLSet() set.StringSet {
	nonResourceURLs := set.NewStringSet()
	for _, rule := range p.granted {
		for _, url := range rule.GetNonResourceUrls() {
			nonResourceURLs.Add(url)
		}
	}
	return nonResourceURLs
}

// ToSlice returns a sorted list of policy rules with the rules broken up by apiGroup and resource.
func (p *policyRuleSet) ToSlice() []*storage.PolicyRule {
	if len(p.granted) == 0 {
		return nil
	}
	return p.granted
}

// GetPermissionSet returns a map of verbs and corresponding resources in the policy rule set
func (p *policyRuleSet) GetPermissionMap() map[string]set.StringSet {
	permissionSet := make(map[string]set.StringSet)
	for _, rule := range p.granted {
		if rule.GetResources() == nil {
			continue
		}
		for _, verb := range rule.GetVerbs() {
			if permissionSet[verb].Contains("*") {
				continue
			}

			for _, resource := range rule.GetResources() {
				if resource == "*" {
					permissionSet[verb] = set.NewStringSet(resource)
					break
				}

				resourceSet := permissionSet[verb]
				resourceSet.Add(resource)
				permissionSet[verb] = resourceSet
			}
		}
	}
	return permissionSet
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
