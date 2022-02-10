package service

import (
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/reprocessor"
	vulReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(cveDataStore.Singleton(), dackbox.GetIndexQueue(), reprocessor.Singleton(), vulReqMgr.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
