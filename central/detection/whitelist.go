package detection

import (
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
)

func matchesDeploymentWhitelists(deployment *storage.Deployment, policy *storage.Policy) bool {
	for _, whitelist := range policy.GetWhitelists() {
		if matchesDeploymentWhitelist(deployment, whitelist) {
			return true
		}
	}
	return false
}

func matchesDeploymentWhitelist(deployment *storage.Deployment, whitelist *storage.Whitelist) bool {
	if whitelistIsExpired(whitelist) {
		return false
	}
	deploymentWhitelist := whitelist.GetDeployment()
	if deploymentWhitelist == nil {
		return false
	}
	if deploymentWhitelist.GetScope() != nil && !scopecomp.WithinScope(deploymentWhitelist.GetScope(), deployment) {
		return false
	}
	if deploymentWhitelist.GetName() != "" && deploymentWhitelist.GetName() != deployment.GetName() {
		return false
	}
	return true
}

func matchesImageWhitelist(image string, policy *storage.Policy) bool {
	for _, w := range policy.GetWhitelists() {
		if w.GetImage() == nil {
			continue
		}
		if whitelistIsExpired(w) {
			continue
		}
		// The rationale for using a prefix is that it is the easiest way in the current format
		// to support whitelisting registries, registry/remote, etc
		if strings.HasPrefix(image, w.GetImage().GetName()) {
			return true
		}
	}
	return false
}

func whitelistIsExpired(whitelist *storage.Whitelist) bool {
	// If they don't set an expiration time, the whitelist never expires.
	if whitelist.GetExpiration() == nil {
		return false
	}
	return whitelist.GetExpiration().Compare(types.TimestampNow()) < 0
}
