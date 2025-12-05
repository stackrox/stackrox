package deploymentcve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	cveView CveView
)

// NewCveView returns the interface CveView
// that provides querying deployment CVEs stored in the database.
func NewCveView(db postgres.DB) CveView {
	return &cveViewImpl{
		db:     db,
		schema: schema.ImageCvesV2Schema,
	}
}

// Singleton provides the interface to query deployment CVEs stored in the database.
func Singleton() CveView {
	once.Do(func() {
		cveView = NewCveView(globaldb.GetPostgres())
	})
	return cveView
}
