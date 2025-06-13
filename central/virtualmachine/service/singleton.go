package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/virtualmachine/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		datastore.Singleton(),
		sachelper.NewClusterSacHelper(clusterDataStore.Singleton()),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
