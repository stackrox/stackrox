package service

import (
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton Service
)

func initialize() {
	singleton = New(connection.ManagerSingleton())
}

// Singleton returns the singleton instance to use.
func Singleton() Service {
	once.Do(initialize)
	return singleton
}
