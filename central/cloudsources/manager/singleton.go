package manager

import (
	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	m    Manager
)

// Manager for detecting clusters from cloud sources.
//
//go:generate mockgen-wrapper
type Manager interface {
	Start()
	Stop()

	// ShortCircuit will signal the manager to short circuit the collection of clusters based on cloud sources.
	ShortCircuit()
}

// Singleton creates a singleton instance of the cloud sources Manager.
func Singleton() Manager {
	if !features.CloudSources.Enabled() {
		return nil
	}

	once.Do(func() {
		m = newManager(cloudSourcesDS.Singleton(), discoveredClustersDS.Singleton())
	})
	return m
}
