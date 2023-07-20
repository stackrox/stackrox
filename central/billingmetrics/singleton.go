package billingmetrics

import (
	store "github.com/stackrox/rox/central/billingmetrics/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(func() {
		svc = New(store.Singleton())
	})
	return svc
}
