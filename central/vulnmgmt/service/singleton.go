package service

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	imageDS "github.com/stackrox/rox/central/imagev2/datastore/mapper/datastore"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(globaldb.GetPostgres(), deploymentDS.Singleton(), imageDS.Singleton(), podDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
