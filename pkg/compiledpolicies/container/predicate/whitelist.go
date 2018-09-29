package predicate

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, NewWhitelistPredicate)
}

// NewWhitelistPredicate true if the container does not match any whitelists.
func NewWhitelistPredicate(policy *v1.Policy) (Predicate, error) {
	var predicate Predicate
	for _, whitelist := range policy.GetWhitelists() {
		// Only compile container whitelists which have not expired.
		if whitelist.GetContainer() != nil && !utils.WhitelistIsExpired(whitelist) {
			wrap := &whitelistWrapper{whitelist: whitelist.GetContainer()}
			predicate = predicate.And(wrap.shouldProcess)
		}
	}
	return predicate, nil
}

type whitelistWrapper struct {
	whitelist *v1.Whitelist_Container
}

func (w *whitelistWrapper) shouldProcess(container *v1.Container) bool {
	return !MatchesWhitelist(w.whitelist, container)
}

// MatchesWhitelist returns if the given container matches the given whitelist.
func MatchesWhitelist(whitelist *v1.Whitelist_Container, container *v1.Container) bool {
	whitelistName := whitelist.GetImageName()
	containerName := container.GetImage().GetName()

	if whitelistName.GetRegistry() != "" && whitelistName.GetRegistry() != containerName.GetRegistry() {
		return false
	}
	if whitelistName.GetRemote() != "" && whitelistName.GetRemote() != containerName.GetRemote() {
		return false
	}
	if whitelistName.GetTag() != "" && whitelistName.GetTag() != containerName.GetTag() {
		return false
	}
	return true
}
