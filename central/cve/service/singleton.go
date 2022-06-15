package service

import (
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	vulReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		return
	}
	imageCVEDataStore := cveDataStore.Singleton()

	// TODO: Attach other CVE stores.
	as = New(imageCVEDataStore, dackbox.GetIndexQueue(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
