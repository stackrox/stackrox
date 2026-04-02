// Package gc provides a scheduled garbage-collection job that removes
// CVE rows from the cves table when they are no longer referenced by
// any component_cve_edges row.
package gc

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/image/v2/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"gopkg.in/robfig/cron.v2"
)

const (
	// gcBatchSize is the maximum number of orphaned CVEs deleted per batch.
	gcBatchSize = 1000
	// gcMaxBatches is the maximum number of batches processed in one GC run.
	gcMaxBatches = 100
	// gcSchedule is the cron expression for nightly GC runs.
	gcSchedule = "@daily"
)

var (
	log = logging.LoggerForModule()
)

// Manager runs periodic CVE garbage collection.
type Manager struct {
	datastore datastore.DataStore
}

// New returns a new GC Manager.
func New(ds datastore.DataStore) *Manager {
	return &Manager{datastore: ds}
}

// RunOnce executes one full GC sweep, deleting orphaned CVE rows in batches.
// It processes at most gcMaxBatches * gcBatchSize rows per invocation.
// Returns the total number of orphaned CVE rows deleted.
func (m *Manager) RunOnce(ctx context.Context) (int64, error) {
	var totalDeleted int64
	for i := 0; i < gcMaxBatches; i++ {
		n, err := m.datastore.DeleteOrphanedCVEsBatch(ctx, gcBatchSize)
		if err != nil {
			return totalDeleted, errors.Wrap(err, "CVE GC batch deletion failed")
		}
		totalDeleted += n
		if n == 0 {
			break // No more orphans to delete.
		}
	}
	return totalDeleted, nil
}

// Start registers the GC job with the provided cron scheduler and starts it.
// The scheduler is responsible for calling Stop() when shutting down.
func (m *Manager) Start(cronScheduler *cron.Cron) error {
	_, err := cronScheduler.AddFunc(gcSchedule, func() {
		ctx := context.Background()
		n, err := m.RunOnce(ctx)
		if err != nil {
			log.Errorf("CVE GC run failed: %v", err)
			return
		}
		log.Infof("CVE GC deleted %d orphaned CVE rows.", n)
	})
	return err
}
