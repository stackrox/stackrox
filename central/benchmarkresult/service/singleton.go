package service

import (
	"sync"

	benchmarkscanStore "github.com/stackrox/rox/central/benchmarkscan/store"
	benchmarkscheduleStore "github.com/stackrox/rox/central/benchmarkschedule/store"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(benchmarkscanStore.Singleton(), benchmarkscheduleStore.Singleton(), notifierProcessor.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
