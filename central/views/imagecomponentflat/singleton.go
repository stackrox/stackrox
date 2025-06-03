package imagecomponentflat

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	imageComponentFlatView ComponentFlatView
)

// NewComponentFlatView returns the interface ComponentFlatView
// that provides searching image components stored in the database.
func NewComponentFlatView(db postgres.DB) ComponentFlatView {
	if !features.FlattenCVEData.Enabled() {
		return nil
	}

	return &imageComponentFlatViewImpl{
		db:     db,
		schema: schema.ImageComponentV2Schema,
	}
}

// Singleton provides the interface to search image cves stored in the database.
func Singleton() ComponentFlatView {
	once.Do(func() {
		imageComponentFlatView = NewComponentFlatView(globaldb.GetPostgres())
	})
	return imageComponentFlatView
}
