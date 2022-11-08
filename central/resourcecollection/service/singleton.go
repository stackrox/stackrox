package service

import (
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	if !env.PostgresDatastoreEnabled.BooleanSetting() || !features.ObjectCollections.Enabled() {
		return nil
	}
	once.Do(initialize)
	return as
}
