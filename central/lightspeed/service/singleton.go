package service

import (
	"github.com/stackrox/rox/central/lightspeed/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	svc  Service
)

func initialize() {
	svc = New(datastore.Singleton())
}

// Singleton provides the Service instance.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
