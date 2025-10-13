package pruner

import (
	scanResult "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profile "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compRule "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scan "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	scanSettingBinding "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suites "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
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
		pruner = New(compIntegration.Singleton(), compScanSetting.Singleton(), scanResult.Singleton(), compRule.Singleton(), profile.Singleton(), scanSettingBinding.Singleton(), scan.Singleton(), suites.Singleton())
	})
	return pruner
}
