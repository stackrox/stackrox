package k8srbac

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

const wildcard = "*"

// PolicyRuleSet holds a deduplicating set of PolicyRules. It is meant to represent one global set of permissions.
type PolicyRuleSet interface {
	Add(prs ...*storage.PolicyRule)
	Grants(pr *storage.PolicyRule) bool
	ToSlice() []*storage.PolicyRule
}

// NewPolicyRuleSet returns a new PolicyRuleSet instance.
func NewPolicyRuleSet() PolicyRuleSet {
	return &policyRuleSet{
		policyRuleKeys: make(map[policyRuleKey]struct{}),
	}
}

type policyRuleSet struct {
	policyRuleKeys map[policyRuleKey]struct{}
}

// AddAll adds all of the inputs to the set.
func (p *policyRuleSet) Add(prs ...*storage.PolicyRule) {
	for _, pr := range prs {
		p.add(pr)
	}
	p.removeShadowed()
}

// Grants returns if the set of PolicyRules grants all necessary permissions for the given policy rule.
func (p *policyRuleSet) Grants(pr *storage.PolicyRule) bool {
	// Keys the input rule grants.
	keys := grantedKeys(pr)

	// For each key we need to find at least one key that grants that permission,
	for _, key := range keys {
		if !p.grantsKey(key) {
			return false
		}
	}
	return true
}

// ToSlice returns a sorted list of policy rules with the rules broken up by apiGroup and resource.
func (p *policyRuleSet) ToSlice() []*storage.PolicyRule {
	if len(p.policyRuleKeys) == 0 {
		return nil
	}
	sortedList := toSortedList(p.policyRuleKeys)
	return condenseRules(sortedList)
}

// Add a PolicyRule to the set if it is not already granted by the set.
func (p *policyRuleSet) add(pr *storage.PolicyRule) {
	keys := grantedKeys(pr)
	for _, key := range keys {
		p.policyRuleKeys[key] = struct{}{}
	}
}

// grantsKey returns if the policy set grants the permissions for the given key.
func (p *policyRuleSet) grantsKey(key policyRuleKey) bool {
	// Check if we have a match without considering wildcards.
	if _, exists := p.policyRuleKeys[key]; exists {
		return true
	}
	// Check if a wildcard key shadows it.
	return p.isShadowed(key)
}

// Remove all keys where another key grants a superset of its permissions.
func (p *policyRuleSet) removeShadowed() {
	for key := range p.policyRuleKeys {
		if p.isShadowed(key) {
			delete(p.policyRuleKeys, key)
		}
	}
}

// Return whether the set has a key that grants a superset of the given key's permissions.
func (p *policyRuleSet) isShadowed(key policyRuleKey) bool {
	// Look for possible wildcard matches.
	possibleShadowers := key.possibleShadowers()
	for _, wcKey := range possibleShadowers {
		if _, isShadowed := p.policyRuleKeys[wcKey]; isShadowed {
			return true
		}
	}
	return false
}

// Static helper functions.
///////////////////////////

func toSortedList(policyRulesByKey map[policyRuleKey]struct{}) []policyRuleKey {
	// Sort for stability.
	sortedPolicyRuleKeys := make([]policyRuleKey, 0, len(policyRulesByKey))
	for policyRuleKey := range policyRulesByKey {
		sortedPolicyRuleKeys = append(sortedPolicyRuleKeys, policyRuleKey)
	}
	sort.SliceStable(sortedPolicyRuleKeys, func(idx1, idx2 int) bool {
		return policyRuleKeyIsLess(sortedPolicyRuleKeys[idx1], sortedPolicyRuleKeys[idx2])
	})
	return sortedPolicyRuleKeys
}

func condenseRules(sortedPolicyRuleKeys []policyRuleKey) []*storage.PolicyRule {
	// Combine Rules by api/resource, since the two are a combined key.
	var currentRule *storage.PolicyRule
	policyRules := make([]*storage.PolicyRule, 0)
	for _, key := range sortedPolicyRuleKeys {
		if keyShouldAddVerb(key, currentRule) {
			addVerb(key, currentRule)
			continue
		}
		currentRule = ruleFromKey(key)
		policyRules = append(policyRules, currentRule)
	}
	return policyRules
}

func keyShouldAddVerb(key policyRuleKey, rule *storage.PolicyRule) bool {
	if rule == nil {
		return false
	}
	return ruleHasAPIGroup(key.apiGroup, rule) && ruleHasResource(key.resource, rule)
}

func addVerb(key policyRuleKey, rule *storage.PolicyRule) {
	if ruleHasVerb(key.verb, rule) {
		return
	}
	if key.verb == wildcard {
		rule.Verbs = []string{wildcard}
	} else {
		rule.Verbs = append(rule.Verbs, key.verb)
	}
}

func ruleHasAPIGroup(desired string, rule *storage.PolicyRule) bool {
	return sliceutils.StringFind(rule.GetApiGroups(), desired) >= 0
}

func ruleHasResource(desired string, rule *storage.PolicyRule) bool {
	return sliceutils.StringFind(rule.GetResources(), desired) >= 0
}

func ruleHasVerb(desired string, rule *storage.PolicyRule) bool {
	return sliceutils.StringFind(rule.GetVerbs(), desired) >= 0
}

func ruleFromKey(key policyRuleKey) *storage.PolicyRule {
	return &storage.PolicyRule{
		Verbs:     []string{key.verb},
		ApiGroups: []string{key.apiGroup},
		Resources: []string{key.resource},
	}
}

func policyRuleKeyIsLess(pr1, pr2 policyRuleKey) bool {
	if pr1.apiGroup != pr2.apiGroup {
		return pr1.apiGroup < pr2.apiGroup
	}
	if pr1.resource != pr2.resource {
		return pr1.resource < pr2.resource
	}
	return pr1.verb < pr2.verb
}
