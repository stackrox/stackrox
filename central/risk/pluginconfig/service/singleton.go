package service

import (
	"github.com/stackrox/rox/central/risk/scorer/plugin/registry"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

// Singleton returns the singleton instance of the risk scoring plugin config service.
func Singleton() Service {
	once.Do(func() {
		instance = New(registry.Singleton())
	})
	return instance
}
