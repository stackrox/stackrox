package manager

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/central/externalbackups/scheduler"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Manager implements the interface for external backups
//
//go:generate mockgen-wrapper
type Manager interface {
	Upsert(ctx context.Context, backup *storage.ExternalBackup) error
	Test(ctx context.Context, backup *storage.ExternalBackup) error
	Remove(ctx context.Context, id string)

	Backup(ctx context.Context, id string) error
}

type backupInfo struct {
	plugin *storage.ExternalBackup
	backup types.ExternalBackup
}

// New returns a new external backup manager
func New(reporter integrationhealth.Reporter, backupListener listener.BackupListener) Manager {
	return &managerImpl{
		scheduler:            scheduler.New(reporter, backupListener),
		idsToExternalBackups: make(map[string]*backupInfo),
	}
}

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type managerImpl struct {
	scheduler  scheduler.Scheduler
	lock       sync.Mutex
	inProgress concurrency.Flag

	idsToExternalBackups map[string]*backupInfo
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

func (m *managerImpl) Upsert(ctx context.Context, backup *storage.ExternalBackup) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	backupInterface, err := renderExternalBackupFromProto(backup)
	if err != nil {
		return err
	}

	cronTab, err := schedule.ConvertToCronTab(backup.GetSchedule())
	if err != nil {
		return err
	}
	if err := m.scheduler.UpsertBackup(backup.GetId(), cronTab, backupInterface, backup); err != nil {
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	m.idsToExternalBackups[backup.GetId()] = &backupInfo{backup, backupInterface}

	return nil
}

func (m *managerImpl) Test(ctx context.Context, backup *storage.ExternalBackup) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	backupInterface, err := renderExternalBackupFromProto(backup)
	if err != nil {
		return err
	}
	return backupInterface.Test()
}

func (m *managerImpl) getBackup(id string) (*backupInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	backupInfo, ok := m.idsToExternalBackups[id]
	if !ok {
		return nil, fmt.Errorf("backup with id %q does not exist", id)
	}
	return backupInfo, nil
}

func (m *managerImpl) Backup(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if m.inProgress.TestAndSet(true) {
		return errors.New("backup already in progress")
	}

	defer m.inProgress.Set(false)

	bInfo, err := m.getBackup(id)
	if err != nil {
		log.Errorf("unable to run backup: corresponding backup plugin %s not found", id)
		return err
	}

	return m.scheduler.RunBackup(bInfo.backup, bInfo.plugin)
}

func (m *managerImpl) Remove(ctx context.Context, id string) {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil || !ok {
		return
	}

	m.scheduler.RemoveBackup(id)

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.idsToExternalBackups, id)
}
