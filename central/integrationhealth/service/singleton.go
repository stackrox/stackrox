package service

import (
	"github.com/stackrox/stackrox/central/imageintegration"
	"github.com/stackrox/stackrox/central/integrationhealth/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), imageintegration.VulnDefsInfoProvider())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
