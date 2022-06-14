package service

import (
	legacyImageCVEDataStore "github.com/stackrox/stackrox/central/cve/datastore"
	cveDataStore "github.com/stackrox/stackrox/central/cve/image/datastore"
	"github.com/stackrox/stackrox/central/globaldb/dackbox"
	vulReqMgr "github.com/stackrox/stackrox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	var imageCVEDataStore cveDataStore.DataStore
	if features.PostgresDatastore.Enabled() {
		imageCVEDataStore = cveDataStore.Singleton()
	} else {
		imageCVEDataStore = legacyImageCVEDataStore.Singleton()
	}

	// TODO: Attach other CVE stores.
	as = New(imageCVEDataStore, dackbox.GetIndexQueue(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
