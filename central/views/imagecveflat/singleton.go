package imagecveflat

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	imageCVEFlatView CveFlatView
)

// NewCVEFlatView returns the interface CveView
// that provides searching image cves stored in the database.
func NewCVEFlatView(db postgres.DB) CveFlatView {
	cveSchema := schema.ImageCvesV2Schema
	if features.FlattenImageData.Enabled() {
		// image_cves_v2 is replaced by the normalized cves table when FlattenImageData is enabled.
		cveSchema = schema.CvesSchema
	}
	return &imageCVEFlatViewImpl{
		db:     db,
		schema: cveSchema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() CveFlatView {
	once.Do(func() {
		imageCVEFlatView = NewCVEFlatView(globaldb.GetPostgres())
	})
	return imageCVEFlatView
}
