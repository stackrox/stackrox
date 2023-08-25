package service

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
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
		singleton = New(connection.ManagerSingleton(), imageDatastore.Singleton(), riskManager.Singleton())
	})
	return singleton
}
