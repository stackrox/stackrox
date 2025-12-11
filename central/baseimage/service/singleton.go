package service

import (
	"github.com/stackrox/rox/central/baseimage/datastore/repository"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(repository.Singleton(), imageintegration.Set())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
