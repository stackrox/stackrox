package injector

import (
	"context"
	"time"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	productUsageWriteSCC = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))
	log = logging.LoggerForModule()
)

type injectorImpl struct {
	// injector gathers data on tick from this channel.
	tickChan <-chan time.Time
	// onStop is called after injector has stopped the gathering loop.
	onStop         func()
	ds             datastore.DataStore
	stop           concurrency.Signal
	gatherersGroup *sync.WaitGroup
}

func (i *injectorImpl) gather(ctx context.Context) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, productUsageWriteSCC)
	newMetrics, err := i.ds.AggregateAndReset(ctx)
	if err != nil {
		log.Info("Failed to get and reset the aggregated administration usage metrics: ", err)
		return
	}
	if err := i.ds.Add(ctx, newMetrics); err != nil {
		log.Info("Failed to store a usage snapshot: ", err)
	}
}

func (i *injectorImpl) gatherLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	// There will most probably be no data on startup: sensors won't have time
	// to report.
	wg := &sync.WaitGroup{}
	for {
		select {
		case <-i.tickChan:
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
	i.gatherersGroup.Add(1)
	go func() {
		defer i.gatherersGroup.Done()
		i.gatherLoop()
	}()
}

// Stop stops the scheduled timer and wait for the gatherer to stop.
func (i *injectorImpl) Stop() {
	i.stop.Signal()
	i.gatherersGroup.Wait()
	if i.onStop != nil {
		i.onStop()
	}
}
