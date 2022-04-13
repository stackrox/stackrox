package service

import (
	"github.com/stackrox/stackrox/central/group/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(func() {
		svc = New(datastore.Singleton())
	})
	return svc
}
