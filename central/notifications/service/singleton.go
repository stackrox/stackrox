package service

import (
	"github.com/stackrox/rox/central/notifications/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	svc Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	if !features.CentralNotifications.Enabled() {
		return nil
	}
	once.Do(func() {
		svc = newService(datastore.Singleton())
	})
	return svc
}
