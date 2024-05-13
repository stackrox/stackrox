package nodecve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	nodeCVEView CveView
)

// NewCVEView returns the interface CveView
// that provides searching node cves stored in the database.
func NewCVEView(db postgres.DB) CveView {
	return &nodeCVECoreViewImpl{
		db:     db,
		schema: schema.NodeCvesSchema,
	}
}

// Singleton provides the interface to search node cves stored in the database.
func Singleton() CveView {
	once.Do(func() {
		nodeCVEView = NewCVEView(globaldb.GetPostgres())
	})
	return nodeCVEView
}
