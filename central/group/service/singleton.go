package service

import (
	"github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/pkg/sync"
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
