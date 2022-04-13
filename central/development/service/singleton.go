package service

import (
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	singleton Service
	once      sync.Once
)

// Singleton returns the singleton.
func Singleton() Service {
	once.Do(func() {
		singleton = New(connection.ManagerSingleton())
	})
	return singleton
}
