package injector

import (
	"time"

	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
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
	return &injectorImpl{
		ds:             ds,
		gatherersGroup: &sync.WaitGroup{},
	}
}
