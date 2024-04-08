package platformcve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	platformCVEView CveView
)

// NewCVEView returns the interface CveView
// that provides searching platform cves stored in the database.
func NewCVEView(db postgres.DB) CveView {
	return &platformCVECoreViewImpl{
		db:     db,
		schema: schema.ClusterCvesSchema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() CveView {
	once.Do(func() {
		platformCVEView = NewCVEView(globaldb.GetPostgres())
	})
	return platformCVEView
}
