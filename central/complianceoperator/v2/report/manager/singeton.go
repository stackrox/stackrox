package manager

import (
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(scanConfigurationDS.Singleton())
}
