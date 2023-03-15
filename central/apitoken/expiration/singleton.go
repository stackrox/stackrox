package expiration

import (
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	notifier TokenExpirationLoop
	once     sync.Once
)

// Singleton returns the global instance of the expiring API Token notifier loop
func Singleton() TokenExpirationLoop {
	once.Do(func() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			notifier = newExpirationNotifier(datastore.Singleton())
		}
	})
	return notifier
}
