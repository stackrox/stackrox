package certgen

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	siStore "github.com/stackrox/stackrox/central/serviceidentities/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

// ServiceSingleton returns the singleton instance of the certgen service.
func ServiceSingleton() Service {
	once.Do(func() {
		instance = NewService(clusterDataStore.Singleton(), siStore.Singleton())
	})
	return instance
}
