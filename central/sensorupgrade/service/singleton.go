package service

import (
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once      sync.Once
	singleton Service
)

func initialize() {
	var err error
	singleton, err = New(datastore.Singleton(), connection.ManagerSingleton())
	utils.CrashOnError(err)
}

// Singleton returns the singleton instance to use.
func Singleton() Service {
	once.Do(initialize)
	return singleton
}
