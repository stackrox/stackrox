package scheduler

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/export"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"gopkg.in/robfig/cron.v2"
)

var (
	log = logging.LoggerForModule()
)

// Scheduler maintains the schedules for backups
type Scheduler interface {
	UpsertBackup(id, spec string, backup types.ExternalBackup) error
	RemoveBackup(id string)

	RunBackup(backup types.ExternalBackup) error
}

type scheduler struct {
	lock sync.Mutex

	cron              *cron.Cron
	pluginsToEntryIDs map[string]cron.EntryID
}

// New instantiates a new cron scheduler and accounts for adding and removing external backups
func New() Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	return &scheduler{
		pluginsToEntryIDs: make(map[string]cron.EntryID),
		cron:              cronScheduler,
	}
}

func (s *scheduler) backup(w *io.PipeWriter) {
	err := export.Backup(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB(), w, false)
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

func (s *scheduler) send(r io.Reader, backup types.ExternalBackup) error {
	if err := backup.Backup(r); err != nil {
		return errors.Wrapf(err, "failed to send backup to %T", backup)
	}
	return nil
}

func (s *scheduler) RunBackup(backup types.ExternalBackup) error {
	pr, pw := io.Pipe()
	go s.backup(pw)
	if err := s.send(pr, backup); err != nil {
		return err
	}
	log.Infof("Successfully ran backup to %T", backup)
	return nil
}

func (s *scheduler) backupClosure(backup types.ExternalBackup) func() {
	return func() {
		if err := s.RunBackup(backup); err != nil {
			log.Error(err)
		}
	}
}

func (s *scheduler) UpsertBackup(id string, spec string, backup types.ExternalBackup) error {
	entryID, err := s.cron.AddFunc(spec, s.backupClosure(backup))
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
