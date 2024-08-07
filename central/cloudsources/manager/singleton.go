package manager

import (
	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	m    Manager
)

// Manager for fetching clusters from cloud sources.
//
//go:generate mockgen-wrapper
type Manager interface {
	Start()
	Stop()

	// ShortCircuit signals the manager to short circuit the collection of clusters from cloud sources.
	ShortCircuit()

	// MarkClusterSecured is received when a new secured cluster is added and marks any discovered clusters
	// associated with the cluster ID as secured.
	MarkClusterSecured(id string)

	// MarkClusterUnsecured is received when a secured cluster is removed and marks any discovered clusters
	// associated with the cluster ID as unsecured
	MarkClusterUnsecured(id string)
}

// Singleton creates a singleton instance of the cloud sources Manager.
func Singleton() Manager {
	once.Do(func() {
		m = newManager(cloudSourcesDS.Singleton(), discoveredClustersDS.Singleton(), clusterDS.Singleton())
	})
	return m
}
