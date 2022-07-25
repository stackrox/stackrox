package service

import (
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	singleton Service
	once      sync.Once
)

// Singleton returns the singleton.
func Singleton() Service {
	once.Do(func() {
		singleton = New(connection.ManagerSingleton(), imageDataStore.Singleton())
	})
	return singleton
}
