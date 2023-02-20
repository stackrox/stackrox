package imagecve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	imageCVEView CveView
)

// NewCVEView returns the interface CveView
// that provides searching image cves stored in the database.
func NewCVEView(db *postgres.DB) CveView {
	return &imageCVECoreViewImpl{
		db:     db,
		schema: schema.ImageCvesSchema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() CveView {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil
	}

	once.Do(func() {
		imageCVEView = NewCVEView(globaldb.GetPostgres())
	})
	return imageCVEView
}
