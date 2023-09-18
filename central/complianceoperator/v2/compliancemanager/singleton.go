package compliancemanager

import (
	integrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	manager Manager
	once    sync.Once
)

// Singleton returns the compliance operator manager
func Singleton() Manager {
	once.Do(func() {
		manager = New(connection.ManagerSingleton(), integrationDS.Singleton())
	})
	return manager
}
