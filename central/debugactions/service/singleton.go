package service

import (
	"github.com/stackrox/rox/central/debugactions"
	"github.com/stackrox/rox/central/debugactions/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	das Service
)

func initialize() {
	das = New(manager.Singleton())
}

// Singleton returns the sole instance of the Service
func Singleton() Service {
	if !debugactions.DebugActions.BooleanSetting() {
		return nil
	}
	once.Do(initialize)
	return das
}
