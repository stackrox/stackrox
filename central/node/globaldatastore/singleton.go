package globaldatastore

import (
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	"github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	globalDataStoreInstance     GlobalDataStore
	initGlobalDataStoreInstance sync.Once
)

// Singleton returns the singleton global node datastore instance.
func Singleton() GlobalDataStore {
	initGlobalDataStoreInstance.Do(func() {
		var err error
		if features.HostScanning.Enabled() {
			globalDataStoreInstance, _ = globaldatastore.New(datastore.Singleton())
		} else {
			globalDataStoreInstance, err = New(globalstore.Singleton(), index.New(globalindex.GetGlobalTmpIndex()), riskDS.Singleton(), ranking.NodeRanker(), ranking.ComponentRanker())
		}
		utils.Must(err)
	})
	return globalDataStoreInstance
}
