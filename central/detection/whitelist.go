package detection

import (
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

func matchesDeploymentWhitelists(deployment *storage.Deployment, whitelists []*compiledWhitelist) bool {
	for _, whitelist := range whitelists {
		if whitelist.MatchesDeployment(deployment) {
			return true
		}
	}
	return false
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
