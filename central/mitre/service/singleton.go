package service

import (
	"github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton())
}

// Singleton provides the instance of the MITRE ATTACK Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
