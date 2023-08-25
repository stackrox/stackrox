package compliancemanager

import (
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
		manager = New(connection.ManagerSingleton())
	})
	return manager
}
