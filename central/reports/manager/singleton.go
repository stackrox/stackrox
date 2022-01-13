package manager

import (
	"github.com/stackrox/rox/central/reports/scheduler"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager

	log = logging.LoggerForModule()
)

func initialize() {
	instance = &managerImpl{
		scheduler: scheduler.New(),
	}
}

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	if !features.VulnReporting.Enabled() {
		return nil
	}
	once.Do(initialize)
	return instance
}
