package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/role/store"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(store.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
