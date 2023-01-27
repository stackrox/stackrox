package expiration

import (
	"github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	notifier ExpiryNotifier
	once     sync.Once
)

// Singleton ...
func Singleton() ExpiryNotifier {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil
	}
	once.Do(func() {
		notifier = newExpirationNotifier(datastore.Singleton())
	})
	return notifier
}
