package service

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(deploymentDS.Singleton(), imageDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
