package service

import (
	"sync"

	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/notifier/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), processor.Singleton(), detection.GetDetector())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
