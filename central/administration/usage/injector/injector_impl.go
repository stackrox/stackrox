package injector

import (
	"context"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/periodicworker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	administrationUsageUsageWriteSCC = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))
	log = logging.LoggerForModule()
)

type injectorImpl struct {
	ds             datastore.DataStore
	worker         *periodicworker.Worker
	gatherersGroup *sync.WaitGroup
}

func (i *injectorImpl) gather(ctx context.Context) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, administrationUsageUsageWriteSCC)
	newMetrics, err := i.ds.AggregateAndReset(ctx)
	if err != nil {
		log.Info("Failed to get and reset the aggregated administration usage metrics: ", err)
		return
	}
	if err := i.ds.Add(ctx, newMetrics); err != nil {
		log.Info("Failed to store a usage snapshot: ", err)
	}
}

// Start initiates periodic data injections to the database with the
// collected usage.
func (i *injectorImpl) Start() {
	i.worker = periodicworker.MustNew(periodicworker.Config{
		Name:       "administration_usage_gather",
		Interval:   aggregationPeriod,
		RunOnStart: false,
		Task: func(ctx context.Context) error {
			var wg sync.WaitGroup
			wg.Go(func() {
				i.gather(ctx)
			})
			wg.Wait()
			return nil
		},
		Logger: log,
	})

	i.gatherersGroup.Go(func() {
		i.worker.Start()
		<-i.worker.Stopped().Done()
		log.Info("Usage reporting stopped")
	})
}

// Stop stops the scheduled timer and wait for the gatherer to stop.
func (i *injectorImpl) Stop() {
	if i.worker != nil {
		i.worker.Stop()
	}
	i.gatherersGroup.Wait()
}
