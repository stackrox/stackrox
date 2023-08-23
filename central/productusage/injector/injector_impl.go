package injector

import (
	"context"
	"time"

	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

const aggregationPeriod = 1 * time.Hour

var (
	productUsageWriteSCC = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))
	log = logging.LoggerForModule()
)

type injectorImpl struct {
	ds   datastore.DataStore
	stop concurrency.Signal
}

func (i *injectorImpl) gather(ctx context.Context) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, productUsageWriteSCC)
	newMetrics, err := i.ds.AggregateAndReset(ctx)
	if err != nil {
		log.Info("Failed to get and reset the aggregated product usage metrics: ", err)
		return
	}
	if err := i.ds.Add(ctx, newMetrics); err != nil {
		log.Info("Failed to store a usage snapshot: ", err)
	}
}

func (i *injectorImpl) gatherLoop() {
	ticker := time.NewTicker(aggregationPeriod)
	defer ticker.Stop()
	// There will most probably be no data on startup: sensors won't have time
	// to report.
	wg := &sync.WaitGroup{}
	for {
		select {
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				i.gather(ctx)
			}()
		case <-i.stop.Done():
			cancel()
			wg.Wait()
			log.Info("Usage reporting stopped")
			i.stop.Reset()
			return
		}
	}
}

// Start initiates periodic data injections to the database with the
// collected usage.
func (i *injectorImpl) Start() {
	go i.gatherLoop()
}

// Stop stops the scheduled timer
func (i *injectorImpl) Stop() {
	i.stop.Signal()
}
