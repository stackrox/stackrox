package images

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	imageView ImageView
)

// NewImageView returns the interface ImageView
// that provides methods for searching images stored in the database.
func NewImageView(db postgres.DB) ImageView {
	return &imageCoreViewImpl{
		db:     db,
		schema: schema.ImagesSchema,
	}
}

// Singleton provides the interface to search images stored in the database.
func Singleton() ImageView {
	once.Do(func() {
		imageView = NewImageView(globaldb.GetPostgres())
	})
	return imageView
}
