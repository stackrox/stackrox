package manager

import (
	"github.com/stackrox/rox/central/debugactions"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	manager Manager
)

func initialize() {
	manager = New()
	manager.Start()
}

// Singleton returns the sole instance of the Manager
func Singleton() Manager {
	if !debugactions.DebugActions.BooleanSetting() {
		return nil
	}
	once.Do(initialize)
	return manager
}
