package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

type policyRuleKey struct {
	verb     string
	apiGroup string
	resource string
}

// Returns the keys that grant at least as much permission as this key.
func (p *policyRuleKey) possibleShadowers() []policyRuleKey {
	keys := make([]policyRuleKey, 0, 8)
	for _, verb := range []string{p.verb, wildcard} {
		for _, resource := range []string{p.resource, wildcard} {
			for _, apiGroup := range []string{p.apiGroup, wildcard} {
				key := policyRuleKey{
					verb:     verb,
					apiGroup: apiGroup,
					resource: resource,
				}
				// Skip any key == p since a key cannot shadow itself.
				if key != *p {
					keys = append(keys, key)
				}
			}
		}
	}
	return keys
}

// grantedKeys breaks up a PolicyRule into a set of individual permissions that can be compared.
func grantedKeys(pr *storage.PolicyRule) []policyRuleKey {
	keys := make([]policyRuleKey, 0, len(pr.GetVerbs())*len(pr.GetApiGroups())*len(pr.GetResources()))
	for _, verb := range pr.GetVerbs() {
		for _, resource := range pr.GetResources() {
			for _, apiGroup := range pr.GetApiGroups() {
				key := policyRuleKey{
					verb:     verb,
					apiGroup: apiGroup,
					resource: resource,
				}
				keys = append(keys, key)
			}
		}
	}
	return keys
}
