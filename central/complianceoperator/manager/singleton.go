package manager

import (
	complianceDatastore "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/standards"
	checkResultsDatastore "github.com/stackrox/stackrox/central/complianceoperator/checkresults/datastore"
	profileDatastore "github.com/stackrox/stackrox/central/complianceoperator/profiles/datastore"
	rulesDatastore "github.com/stackrox/stackrox/central/complianceoperator/rules/datastore"
	scansDatastore "github.com/stackrox/stackrox/central/complianceoperator/scans/datastore"
	scanSettingBindingDatastore "github.com/stackrox/stackrox/central/complianceoperator/scansettingbinding/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	manager Manager
	once    sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Manager {
	once.Do(func() {
		var err error
		manager, err = NewManager(standards.RegistrySingleton(), profileDatastore.Singleton(), scansDatastore.Singleton(), scanSettingBindingDatastore.Singleton(), rulesDatastore.Singleton(), checkResultsDatastore.Singleton(), complianceDatastore.Singleton())
		utils.Must(err)
	})
	return manager
}
