package alertmanager

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/detection/runtime"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
)

var (
	once sync.Once

	alertManager AlertManager
)

func initialize() {
	alertManager = New(notifierProcessor.Singleton(), alertDataStore.Singleton(), runtime.SingletonDetector())
}

// Singleton returns the singleton instance of an AlertManager
func Singleton() AlertManager {
	once.Do(initialize)
	return alertManager
}
