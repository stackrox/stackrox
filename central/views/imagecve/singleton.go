package imagecve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
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
func NewCVEView(db postgres.DB) CveView {
	cveSchema := schema.ImageCvesSchema
	if features.FlattenCVEData.Enabled() {
		cveSchema = schema.ImageCvesV2Schema
	}

	return &imageCVECoreViewImpl{
		db:     db,
		schema: cveSchema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() CveView {
	once.Do(func() {
		imageCVEView = NewCVEView(globaldb.GetPostgres())
	})
	return imageCVEView
}
