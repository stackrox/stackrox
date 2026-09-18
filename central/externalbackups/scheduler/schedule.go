package scheduler

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/export"
	"github.com/stackrox/rox/central/systeminfo/listener"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"gopkg.in/robfig/cron.v2"
)

var (
	log = logging.LoggerForModule()
)

// Scheduler maintains the schedules for backups
type Scheduler interface {
	UpsertBackup(id, spec string, backup types.ExternalBackup, plugin *storage.ExternalBackup) error
	RemoveBackup(id string)

	RunBackup(backup types.ExternalBackup, plugin *storage.ExternalBackup) error
}

type scheduler struct {
	lock              sync.Mutex
	cron              *cron.Cron
	reporter          integrationhealth.Reporter
	backupListener    listener.BackupListener
	pluginsToEntryIDs map[string]cron.EntryID
}

// New instantiates a new cron scheduler and accounts for adding and removing external backups
func New(reporter integrationhealth.Reporter, backupListener listener.BackupListener) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	s := &scheduler{
		pluginsToEntryIDs: make(map[string]cron.EntryID),
		cron:              cronScheduler,
		reporter:          reporter,
		backupListener:    backupListener,
	}

	// Register with the background worker registry for debug endpoint
	// observability. The scheduler does not fit a standard archetype: it
	// dynamically adds/removes robfig/cron jobs rather than owning a single
	// periodic loop.
	backgroundworker.Global.Register(&backgroundworker.StatusAdapter{
		WorkerName: "external-backup-scheduler",
		WorkerKind: "cron-scheduler",
		StatusFunc: func() backgroundworker.WorkerStatus {
			s.lock.Lock()
			activeBackups := len(s.pluginsToEntryIDs)
			s.lock.Unlock()

			extra := map[string]any{
				"active_backups": activeBackups,
			}
			if entries := s.cron.Entries(); len(entries) > 0 && !entries[0].Next.IsZero() {
				extra["next_run_time"] = entries[0].Next.UTC().Format(time.RFC3339)
			}

			return backgroundworker.WorkerStatus{
				State: "running",
				Extra: extra,
			}
		},
	})

	return s
}

func (s *scheduler) backup(w *io.PipeWriter, includeCerts bool) {
	err := export.BackupPostgres(context.Background(), globaldb.GetPostgres(), s.backupListener, includeCerts, w)
	if err != nil {
		log.Errorf("Failed to write backup to io.writer: %v", err)
		if err := w.CloseWithError(err); err != nil {
			log.Errorf("Could not close writer for backup with error: %v", err)
		}
		return
	}
	if err := w.Close(); err != nil {
		log.Errorf("Error closing writer for backup: %v", err)
	}
}

func (s *scheduler) send(r io.ReadCloser, backup types.ExternalBackup) error {
	if err := backup.Backup(r); err != nil {
		return errors.Wrapf(err, "failed to send backup to %T", backup)
	}
	return nil
}

func (s *scheduler) RunBackup(backup types.ExternalBackup, plugin *storage.ExternalBackup) error {
	pr, pw := io.Pipe()

	// Include certificates in backup by default.
	includeCerts := plugin.GetIncludeCertificatesOpt() == nil || plugin.GetIncludeCertificates()
	go s.backup(pw, includeCerts)

	err := s.send(pr, backup)

	if err != nil {
		s.reporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
			Id:            plugin.GetId(),
			Name:          plugin.GetName(),
			Type:          storage.IntegrationHealth_BACKUP,
			Status:        storage.IntegrationHealth_UNHEALTHY,
			LastTimestamp: protocompat.TimestampNow(),
			ErrorMessage:  err.Error(),
		})
		return err
	}
	s.reporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
		Id:            plugin.GetId(),
		Name:          plugin.GetName(),
		Type:          storage.IntegrationHealth_BACKUP,
		Status:        storage.IntegrationHealth_HEALTHY,
		LastTimestamp: protocompat.TimestampNow(),
		ErrorMessage:  "",
	})
	log.Infof("Successfully ran backup to %T", backup)
	return nil
}

func (s *scheduler) backupClosure(backup types.ExternalBackup, plugin *storage.ExternalBackup) func() {
	return func() {
		if err := s.RunBackup(backup, plugin); err != nil {
			log.Error(err)
		}
	}
}

func (s *scheduler) UpsertBackup(id string, spec string, backup types.ExternalBackup, plugin *storage.ExternalBackup) error {
	entryID, err := s.cron.AddFunc(spec, s.backupClosure(backup, plugin))
	if err != nil {
		return err
	}

	// Remove the old entry if this is an update
	if oldEntryID, ok := s.pluginsToEntryIDs[id]; ok {
		s.cron.Remove(oldEntryID)
	}
	s.pluginsToEntryIDs[id] = entryID
	return nil
}

func (s *scheduler) RemoveBackup(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldEntryID := s.pluginsToEntryIDs[id]
	s.cron.Remove(oldEntryID)
	delete(s.pluginsToEntryIDs, id)
}
