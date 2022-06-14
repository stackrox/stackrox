package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	as   DataStore
	log  = logging.LoggerForModule()
)

// Singleton creates a singleton for the sac datastore and loads the plugin client config
func Singleton() DataStore {
	once.Do(func() {
		var err error
		cliMgr := sac.AuthPluginClientManagerSingleton()
		as, err = New(globaldb.GetGlobalDB(), cliMgr)
		if err != nil {
			log.Panic(err)
		}
	})
	return as
}
