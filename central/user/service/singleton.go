package service

import (
	"github.com/stackrox/rox/central/user/store"
	"github.com/stackrox/rox/pkg/sync"
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
