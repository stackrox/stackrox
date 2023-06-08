package standards

import (
	"github.com/stackrox/rox/central/compliance/framework"
	pgControl "github.com/stackrox/rox/central/compliance/standards/control"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	pgStandard "github.com/stackrox/rox/central/compliance/standards/standard"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	registryInstance     *Registry
	registryInstanceInit sync.Once
)

// RegistrySingleton returns the singleton instance of the compliance standards Registry.
func RegistrySingleton() *Registry {
	registryInstanceInit.Do(func() {
		standardStore := pgStandard.New(globaldb.GetPostgres())
		standardIndexer := pgStandard.NewIndexer(globaldb.GetPostgres())

		controlStore := pgControl.New(globaldb.GetPostgres())
		controlIndexer := pgControl.NewIndexer(globaldb.GetPostgres())

		var err error
		registryInstance, err = NewRegistry(standardStore, standardIndexer, controlStore, controlIndexer, framework.RegistrySingleton(), metadata.AllStandards...)
		utils.CrashOnError(err)
	})
	return registryInstance
}
