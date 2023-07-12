package billingmetrics

import (
	bmstore "github.com/stackrox/rox/central/billingmetrics/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(func() {
		svc = New(bmstore.New(globaldb.GetPostgres()))
	})
	return svc
}
