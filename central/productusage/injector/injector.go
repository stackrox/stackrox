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

const period = 1 * time.Hour

var (
	metricsWriter = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))

	once     sync.Once
	injector Injector
	log      = logging.LoggerForModule()
)

func (i *injectorImpl) gather(ctx context.Context) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, metricsWriter)
	newMetrics, err := i.ds.AggregateAndFlush(ctx)
	if err != nil {
		log.Debug("Failed to get and flush the aggregated usage metrics: ", err)
		return
	}

	// Store the average values to smooth short (< 2 periods) peaks and drops.
	averageMetrics := average(i.previousMetrics, newMetrics)
	averageMetrics.Timestamp = newMetrics.Timestamp
	if err := i.ds.Insert(ctx, averageMetrics); err != nil {
		log.Debug("Failed to store a usage snapshot: ", err)
	}
	i.previousMetrics = newMetrics
}

func (i *injectorImpl) gatherLoop() {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	i.gather(ctx)
	for {
		select {
		case <-ticker.C:
			i.gather(ctx)
		case <-i.stop.Done():
			cancel()
			log.Info("Usage reporting stopped")
			i.stop.Reset()
			return
		}
	}
}

// Injector is the usage metrics injector interface.
type Injector interface {
	Start()
	Stop()
}

type injectorImpl struct {
	ds   datastore.DataStore
	stop concurrency.Signal

	previousMetrics *storage.SecuredUnits
}

// NewInjector creates an injector instance.
func NewInjector(ds datastore.DataStore) Injector {
	return &injectorImpl{
		ds:              ds,
		stop:            concurrency.NewSignal(),
		previousMetrics: &storage.SecuredUnits{},
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

// Singleton returns the injector singleton.
func Singleton() Injector {
	once.Do(func() {
		injector = NewInjector(datastore.Singleton())
	})
	return injector
}
