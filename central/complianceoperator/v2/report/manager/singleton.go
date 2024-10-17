package manager

import (
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of compliance report Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(scanConfigurationDS.Singleton(), scanDS.Singleton(), profileDatastore.Singleton(), reportGen.Singleton())
}
