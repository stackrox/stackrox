package service

import (
	"sync"

	detection "bitbucket.org/stack-rox/apollo/central/detection/singletons"
	"bitbucket.org/stack-rox/apollo/central/notifier/processor"
	"bitbucket.org/stack-rox/apollo/central/notifier/store"
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
