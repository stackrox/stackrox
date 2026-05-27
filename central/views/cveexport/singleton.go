package cveexport

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	view CveExportView
)

// NewCveExportView returns a CveExportView backed by the given database.
func NewCveExportView(db postgres.DB) CveExportView {
	return &viewImpl{
		db:     db,
		schema: schema.ImageCvesV2Schema,
	}
}

// Singleton provides the CveExportView instance.
func Singleton() CveExportView {
	once.Do(func() {
		view = NewCveExportView(globaldb.GetPostgres())
	})
	return view
}
