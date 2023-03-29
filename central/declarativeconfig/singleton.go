package declarativeconfig

import (
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// ManagerSingleton provides the instance of Manager to use.
func ManagerSingleton(registry authproviders.Registry) Manager {
	once.Do(func() {
		instance = New(
			env.DeclarativeConfigReconcileInterval.DurationSetting(),
			env.DeclarativeConfigWatchInterval.DurationSetting(),
			updater.DefaultResourceUpdaters(registry),
			reporter.Singleton(),
			types.UniversalNameExtractor(),
			types.UniversalIDExtractor(),
		)
	})
	return instance
}
