package service

import "github.com/stackrox/stackrox/pkg/sync"

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New()
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
