package predicate

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
	"github.com/stackrox/rox/pkg/scopecomp"
)

func init() {
	compilers = append(compilers, newWhitelistPredicate)
}

func newWhitelistPredicate(policy *v1.Policy) (Predicate, error) {
	var predicate Predicate
	for _, whitelist := range policy.GetWhitelists() {
		// Only compile deployment whitelists which have not expired.
		if whitelist.GetDeployment() != nil && !utils.WhitelistIsExpired(whitelist) {
			wrap := &whitelistWrapper{whitelist.GetDeployment()}
			predicate = predicate.And(wrap.shouldProcess)
		}
	}
	return predicate, nil
}

type whitelistWrapper struct {
	whitelist *v1.Whitelist_Deployment
}

func (w *whitelistWrapper) shouldProcess(deployment *storage.Deployment) bool {
	return !MatchesWhitelist(w.whitelist, deployment)
}

// MatchesWhitelist returns true if the given deployment matches the given whitelist.
func MatchesWhitelist(whitelist *v1.Whitelist_Deployment, deployment *storage.Deployment) bool {
	if whitelist == nil {
		return false
	}
	if whitelist.GetScope() != nil && !scopecomp.WithinScope(whitelist.GetScope(), deployment) {
		return false
	}
	if whitelist.GetName() != "" && whitelist.GetName() != deployment.GetName() {
		return false
	}
	return true
}
