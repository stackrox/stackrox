package imageintegration

import (
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	is              integration.Set
	vulDefsProvider scanners.VulnDefsInfoProvider
)

func initialize() {
	// This is the set of image integrations currently active, and the ToNotify that updates that set.
	is = integration.NewSet(reporter.Singleton())
	vulDefsProvider = scanners.NewVulnDefsInfoProvider(is.ScannerSet())
}

// Set provides the set of image integrations currently in use by central.
func Set() integration.Set {
	once.Do(initialize)
	return is
}

// VulnDefsInfoProvider provides the vulnerability definitions information provider.
func VulnDefsInfoProvider() scanners.VulnDefsInfoProvider {
	once.Do(initialize)
	return vulDefsProvider
}
