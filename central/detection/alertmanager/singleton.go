package alertmanager

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
	alertManager = New(notifierProcessor.Singleton(), alertDataStore.Singleton())
}

// Singleton returns the singleton instance of an AlertManager
func Singleton() AlertManager {
	once.Do(initialize)
	return alertManager
}
