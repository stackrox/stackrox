package manager

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/sync"

	// Make sure all restore formats are registered
	_ "github.com/stackrox/rox/central/globaldb/v2backuprestore/formats/all"
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
