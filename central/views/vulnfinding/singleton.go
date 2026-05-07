package vulnfinding

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	view FindingView
)

// NewFindingView returns a FindingView backed by the given database.
func NewFindingView(db postgres.DB) FindingView {
	return &viewImpl{
		db:     db,
		schema: schema.ImageCvesV2Schema,
	}
}

// Singleton provides the FindingView instance.
func Singleton() FindingView {
	once.Do(func() {
		view = NewFindingView(globaldb.GetPostgres())
	})
	return view
}
