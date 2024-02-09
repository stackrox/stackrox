package pruner

import (
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	pruner Pruner
	once   sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Pruner {
	once.Do(func() {
		pruner = New(compIntegration.Singleton(), compScanSetting.Singleton())
	})
	return pruner
}
