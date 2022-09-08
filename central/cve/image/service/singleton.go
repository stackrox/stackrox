package service

import (
	legacyCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	vulReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	if !features.PostgresDatastore.Enabled() {
		as = New(legacyCVEDataStore.CVESuppressManager(), vulReqMgr.Singleton())
		return
	}
	as = New(cveDataStore.Singleton(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
