package service

import (
	cveDataStore "github.com/stackrox/stackrox/central/cve/image/datastore"
	vulReqMgr "github.com/stackrox/stackrox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(cveDataStore.Singleton(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
