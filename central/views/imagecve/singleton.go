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

	imageCVEView ImageCVEView
)

// NewGenericImageCVEView returns the interface ImageCVEView
// that provides searching image cves stored in the database.
func NewGenericImageCVEView(db *postgres.DB) ImageCVEView {
	return &imageCVECoreViewImpl{
		db:     db,
		schema: schema.ImageCvesSchema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() ImageCVEView {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil
	}

	once.Do(func() {
		imageCVEView = NewGenericImageCVEView(globaldb.GetPostgres())
	})
	return imageCVEView
}
