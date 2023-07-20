package store

import (
	bmstore "github.com/stackrox/rox/central/billingmetrics/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	s    Store
	once sync.Once
)

// Singleton returns the billing metrics store singleton.
func Singleton() Store {
	once.Do(func() {
		s = bmstore.New(globaldb.GetPostgres())
	})
	return s
}
