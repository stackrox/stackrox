package manager

import (
	"github.com/stackrox/stackrox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/stackrox/pkg/migrations"
	"github.com/stackrox/stackrox/pkg/sync"

	// Make sure all restore formats are registered
	_ "github.com/stackrox/stackrox/central/globaldb/v2backuprestore/formats/all"
)

var (
	managerInstance     Manager
	managerInstanceInit sync.Once
)

// Singleton returns the unique singleton instance of the database backup/restore manager.
func Singleton() Manager {
	managerInstanceInit.Do(func() {
		managerInstance = newManager(migrations.DBMountPath(), formats.RegistrySingleton())
	})
	return managerInstance
}
