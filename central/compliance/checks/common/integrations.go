package common

import (
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
)

func atLeastOneMatches(image *storage.ImageName, integrations []framework.ImageMatcher) bool {
	for _, s := range integrations {
		if s.Match(image) {
			return true
		}
	}
	return false
}

func atLeastOneRegistryAndScannerMatch(image *storage.ImageName, registryIntegrations []framework.ImageMatcher, scannerIntegrations []framework.ImageMatcher) bool {
	return atLeastOneMatches(image, registryIntegrations) && atLeastOneMatches(image, scannerIntegrations)
}

// AllDeployedImagesHaveMatchingIntegrationsInterpretation is the interpretation text for CheckAllDeployedImagesHaveMatchingIntegrations.
const AllDeployedImagesHaveMatchingIntegrationsInterpretation = `StackRox checks that every deployed image has a matching registry and scanner integration configured, so that an accurate component inventory can be maintained.`

// CheckAllDeployedImagesHaveMatchingIntegrations verifies that all deployed images have matching
// registry and scanner integrations.
func CheckAllDeployedImagesHaveMatchingIntegrations(ctx framework.ComplianceContext) {
	var failed bool
	registryIntegrations := ctx.Data().RegistryIntegrations()
	scannerIntegrations := ctx.Data().ScannerIntegrations()

	for _, deployment := range ctx.Data().Deployments() {
		for _, container := range deployment.GetContainers() {
			if !atLeastOneRegistryAndScannerMatch(container.GetImage().GetName(), registryIntegrations, scannerIntegrations) {
				failed = true
				framework.Failf(ctx, "image %s deployed in deployment %s/%s has no matching registry/scanner integration",
					container.GetImage().GetName().GetFullName(), deployment.GetNamespace(), deployment.GetName())
			}
		}
	}
	if !failed {
		framework.Pass(ctx, "All deployed images had matching registry and scanner integrations")
	}

}
