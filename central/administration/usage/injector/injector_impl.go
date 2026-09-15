package injector

import (
	"context"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	administrationUsageUsageWriteSCC = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))
	log = logging.LoggerForModule()
)

type injectorImpl struct {
	ds datastore.DataStore

	worker *backgroundworker.PeriodicWorker
}

func (i *injectorImpl) gather(ctx context.Context) error {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, administrationUsageUsageWriteSCC)
	newMetrics, err := i.ds.AggregateAndReset(ctx)
	if err != nil {
		log.Info("Failed to get and reset the aggregated administration usage metrics: ", err)
		return err
	}
	if err := i.ds.Add(ctx, newMetrics); err != nil {
		log.Info("Failed to store a usage snapshot: ", err)
		return err
	}
	return nil
}

// Start initiates periodic data injections to the database with the
// collected usage.
func (i *injectorImpl) Start() {
	i.worker.Start(context.Background())
}

// Stop stops the scheduled timer and waits for the gatherer to stop.
func (i *injectorImpl) Stop() {
	i.worker.Stop()
}
