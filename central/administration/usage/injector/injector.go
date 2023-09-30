package injector

import (
	"time"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

const aggregationPeriod = 1 * time.Hour

// Injector is the usage metrics injector interface.
type Injector interface {
	Start()
	Stop()
}

// NewInjector creates an injector instance.
func NewInjector(ds datastore.DataStore) Injector {
	ticker := time.NewTicker(aggregationPeriod)
	return &injectorImpl{
		tickChan:       ticker.C,
		onStop:         ticker.Stop,
		ds:             ds,
		stop:           concurrency.NewSignal(),
		gatherersGroup: &sync.WaitGroup{},
	}
}
