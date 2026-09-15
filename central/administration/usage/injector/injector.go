package injector

import (
	"time"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/backgroundworker"
)

const aggregationPeriod = 1 * time.Hour

// Injector is the usage metrics injector interface.
type Injector interface {
	Start()
	Stop()
}

// NewInjector creates an injector instance.
func NewInjector(ds datastore.DataStore) Injector {
	impl := &injectorImpl{
		ds: ds,
	}
	impl.worker = &backgroundworker.PeriodicWorker{
		Name:     "usage-metrics-injector",
		Interval: aggregationPeriod,
		// RunOnStart is intentionally omitted: there will most probably be no
		// data on startup, since sensors won't have had time to report yet.
		// gather runs synchronously in the PeriodicWorker loop. The original
		// implementation spawned each call in a separate goroutine, but
		// sequential execution is safer (no concurrent AggregateAndReset calls).
		Run: impl.gather,
	}
	backgroundworker.Global.Register(impl.worker)
	return impl
}
