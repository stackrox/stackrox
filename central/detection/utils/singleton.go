package utils

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
)

var (
	once sync.Once

	alertManager AlertManager
)

func initialize() {
	alertManager = NewAlertManager(notifierProcessor.Singleton(), alertDataStore.Singleton())
}

// SingletonAlertManager returns the singleton instance of an AlertManager
func SingletonAlertManager() AlertManager {
	once.Do(initialize)
	return alertManager
}
