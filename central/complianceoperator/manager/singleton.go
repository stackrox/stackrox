package manager

import (
	"github.com/stackrox/rox/central/compliance/standards"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	rulesDatastore "github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	scanSettingBindingDatastore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager Manager
	once    sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Manager {
	once.Do(func() {
		var err error
		manager, err = NewManager(standards.RegistrySingleton(), profileDatastore.Singleton(), scanSettingBindingDatastore.Singleton(), rulesDatastore.Singleton())
		utils.Must(err)
	})
	return manager
}
