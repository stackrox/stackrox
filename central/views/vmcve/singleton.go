package vmcve

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	vmCVEView CveView
)

// NewCVEView returns the interface CveView
// that provides searching VM CVEs stored in the database.
func NewCVEView(db postgres.DB) CveView {
	return &vmCVECoreViewImpl{
		db:     db,
		schema: schema.VirtualMachineCvev2Schema,
	}
}

// Singleton provides the interface to search VM CVEs stored in the database.
func Singleton() CveView {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		return nil
	}
	once.Do(func() {
		vmCVEView = NewCVEView(globaldb.GetPostgres())
	})
	return vmCVEView
}
