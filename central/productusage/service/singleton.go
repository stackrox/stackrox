package usage

import (
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once

	log = logging.LoggerForModule()
)

// Singleton returns the usage service singleton.
func Singleton() Service {
	once.Do(func() {
		svc = New(datastore.Singleton())
	})
	return svc
}
