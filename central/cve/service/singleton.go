package service

import (
	legacyImageCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	vulReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	// This service is not used in postgres.
	as = New(legacyImageCVEDataStore.Singleton(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
