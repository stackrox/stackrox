package manager

import (
	"fmt"

	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/central/externalbackups/scheduler"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sync"
)

// Manager implements the interface for external backups
type Manager interface {
	Upsert(backup *storage.ExternalBackup) error
	Test(backup *storage.ExternalBackup) error
	Remove(id string)

	Backup(id string) error
}

// New returns a new external backup manager
func New() Manager {
	return &managerImpl{
		scheduler:            scheduler.New(),
		idsToExternalBackups: make(map[string]types.ExternalBackup),
	}
}

type managerImpl struct {
	scheduler scheduler.Scheduler

	lock                 sync.Mutex
	idsToExternalBackups map[string]types.ExternalBackup
}

func renderExternalBackupFromProto(backup *storage.ExternalBackup) (types.ExternalBackup, error) {
	creator, ok := plugins.Registry[backup.GetType()]
	if !ok {
		return nil, fmt.Errorf("external backup with type %q is not implemented", backup.GetType())
	}

	backupInterface, err := creator(backup)
	if err != nil {
		return nil, err
	}
	return backupInterface, nil
}

func (m *managerImpl) Upsert(backup *storage.ExternalBackup) error {
	backupInterface, err := renderExternalBackupFromProto(backup)
	if err != nil {
		return err
	}

	cronTab, err := schedule.ConvertToCronTab(backup.GetSchedule())
	if err != nil {
		return err
	}

	if err := m.scheduler.UpsertBackup(backup.GetId(), cronTab, backupInterface); err != nil {
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	m.idsToExternalBackups[backup.GetId()] = backupInterface

	return nil
}

func (m *managerImpl) Test(backup *storage.ExternalBackup) error {
	backupInterface, err := renderExternalBackupFromProto(backup)
	if err != nil {
		return err
	}
	return backupInterface.Test()
}

func (m *managerImpl) Backup(id string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	backup, ok := m.idsToExternalBackups[id]
	if !ok {
		return fmt.Errorf("backup with id %q does not exist", id)
	}
	return m.scheduler.RunBackup(backup)
}

func (m *managerImpl) Remove(id string) {
	m.scheduler.RemoveBackup(id)

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.idsToExternalBackups, id)
}
