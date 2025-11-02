package service

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	imageDS "github.com/stackrox/rox/central/imagev2/datastore/mapper/datastore"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(deploymentDS.Singleton(), imageDS.Singleton(), imageV2DS.Singleton(), podDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
