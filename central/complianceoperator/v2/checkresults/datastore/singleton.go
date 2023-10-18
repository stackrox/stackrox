package datastore

import (
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		ds = New(storage)
	})
	return ds
}
