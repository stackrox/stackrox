package service

import (
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	var telemetryGatherer *gatherers.RoxGatherer
	if features.Telemetry.Enabled() {
		telemetryGatherer = gatherers.Singleton()
	}
	as = New(datastore.Singleton(), connection.ManagerSingleton(), telemetryGatherer)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
