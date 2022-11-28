package integration

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// GetMatchingImageIntegrations will return all image integrations that match the given image name.
// In case no matching image integrations are found, an empty array will be returned.
// The resulting image integrations array will be sorted by the registry's hostname.
func GetMatchingImageIntegrations(registries []registryTypes.ImageRegistry,
	imageName *storage.ImageName) []registryTypes.ImageRegistry {
	var matchingIntegrations []registryTypes.ImageRegistry
	for _, registry := range registries {
		if registry.Match(imageName) {
			matchingIntegrations = append(matchingIntegrations, registry)
		}
	}

	sort.Slice(matchingIntegrations, func(i, j int) bool {
		// Note: the Name of ImageRegistry does not reflect the registry hostname used within the integration but a
		// name chosen by the creator. Additionally, we have to trim the HTTP prefixes (http:// & https://).
		if matchingIntegrations[i].Config() == nil {
			return true
		}
		if matchingIntegrations[j].Config() == nil {
			return false
		}
		return urlfmt.TrimHTTPPrefixes(matchingIntegrations[i].Config().RegistryHostname) <
			urlfmt.TrimHTTPPrefixes(matchingIntegrations[j].Config().RegistryHostname)
	})

	return matchingIntegrations
}
