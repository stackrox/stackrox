package service

import (
	"sync"

	"github.com/stackrox/rox/central/group/store"
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
