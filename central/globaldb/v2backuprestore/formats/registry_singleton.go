package formats

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	registryInstance     Registry
	registryInstanceInit sync.Once
)

// RegistrySingleton retrieves the global DB export format registry.
func RegistrySingleton() Registry {
	registryInstanceInit.Do(func() {
		registryInstance = newRegistry()
	})
	return registryInstance
}

// MustRegisterNewFormat creates and registers a new DB export format. If the format could not be registered, this
// function panics.
func MustRegisterNewFormat(name string, fileHandlers ...*common.FileHandlerDesc) {
	format := &ExportFormat{
		name:         name,
		fileHandlers: fileHandlers,
	}
	utils.Must(format.Validate())
	utils.Must(RegistrySingleton().RegisterFormat(format))
}
