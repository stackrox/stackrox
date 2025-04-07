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
	if !features.FlattenCVEData.Enabled() {
		return nil
	}

	return &imageCVEFlatViewImpl{
		db:     db,
		schema: schema.ImageCvesV2Schema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() CveFlatView {
	once.Do(func() {
		imageCVEFlatView = NewCVEFlatView(globaldb.GetPostgres())
	})
	return imageCVEFlatView
}
