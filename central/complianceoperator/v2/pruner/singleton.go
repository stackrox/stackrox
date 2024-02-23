package pruner

import (
	scanResult "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	pruner Pruner
	once   sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Pruner {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(func() {
		pruner = New(compIntegration.Singleton(), compScanSetting.Singleton(), scanResult.Singleton())
	})
	return pruner
}
