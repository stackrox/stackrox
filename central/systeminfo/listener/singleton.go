package listener

import (
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ls BackupListener
)

// Singleton provides the singleton instance of the backup listener interface.
func Singleton() BackupListener {
	once.Do(func() {
		ls = newBackupListener(systemInfoStorage.Singleton())
	})
	return ls
}
