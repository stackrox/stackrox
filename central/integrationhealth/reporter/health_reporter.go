package reporter

import "github.com/stackrox/rox/generated/storage"

// IntegrationHealthReporter is an interface to report integration health
type IntegrationHealthReporter interface {
	UpdateIntegrationHealth(*storage.IntegrationHealth)
}
