package injector

import (
	"sync"

	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Injector is the usage metrics injector interface.
type Injector interface {
	Start()
	Stop()
}

// NewInjector creates an injector instance.
func NewInjector(ds datastore.DataStore) Injector {
	return &injectorImpl{
		ds:             ds,
		stop:           concurrency.NewSignal(),
		gatherersGroup: &sync.WaitGroup{},
	}
}
