package detection

import (
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

// UpdateImageIntegration updates the map of active integrations
func (d *detectorImpl) UpdateImageIntegration(integration *sources.ImageIntegration) {
	d.enricher.UpdateImageIntegration(integration)

	go d.EnrichAndReprocess()
}

// RemoveImageIntegration removes an image integration
func (d *detectorImpl) RemoveImageIntegration(id string) {
	d.enricher.RemoveImageIntegration(id)
}
