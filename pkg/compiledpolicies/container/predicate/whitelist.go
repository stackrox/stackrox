package predicate

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	imageTypes "bitbucket.org/stack-rox/apollo/pkg/images/types"
)

func init() {
	compilers = append(compilers, NewWhitelistPredicate)
}

// NewWhitelistPredicate true if the container does not match any whitelists.
func NewWhitelistPredicate(policy *v1.Policy) (Predicate, error) {
	var predicate Predicate
	for _, whitelist := range policy.GetWhitelists() {
		if whitelist.GetContainer() != nil {
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
	whitelistDigest := imageTypes.NewDigest(whitelistName.GetSha()).Digest()
	containerDigest := imageTypes.NewDigest(containerName.GetSha()).Digest()

	if whitelistName.GetSha() != "" && whitelistDigest != containerDigest {
		return false
	}
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
