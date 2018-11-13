package service

import (
	"sync"

	"github.com/stackrox/rox/central/user/store"
)

var (
	svc  Service
	once sync.Once
)

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(func() {
		svc = New(store.Singleton())
	})
	return svc
}
