package datastore

import (
	edgeStorePkg "github.com/stackrox/rox/central/cve/image/componentcveedge/datastore/store/postgres"
	cveStorePkg "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	componentStorePkg "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	pool := globaldb.GetPostgres()

	cveStore := cveStorePkg.New(pool)
	edgeStore := edgeStorePkg.New(pool)
	componentStore := componentStorePkg.New(pool)

	ds = New(cveStore, edgeStore, componentStore)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
