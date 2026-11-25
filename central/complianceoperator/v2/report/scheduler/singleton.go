package scheduler

import (
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of the compliance report scheduler to use.
func Singleton() Manager {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(scanConfigDS.Singleton())
}
