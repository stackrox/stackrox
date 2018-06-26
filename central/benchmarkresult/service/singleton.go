package service

import (
	"sync"

	benchmarkscanStore "bitbucket.org/stack-rox/apollo/central/benchmarkscan/store"
	benchmarkscheduleStore "bitbucket.org/stack-rox/apollo/central/benchmarkschedule/store"
	notifierProcessor "bitbucket.org/stack-rox/apollo/central/notifier/processor"
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
