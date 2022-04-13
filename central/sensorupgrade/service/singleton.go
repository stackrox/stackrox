package service

import (
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once      sync.Once
	singleton Service
)

func initialize() {
	singleton = New(datastore.Singleton(), connection.ManagerSingleton())
}

// Singleton returns the singleton instance to use.
func Singleton() Service {
	once.Do(initialize)
	return singleton
}
