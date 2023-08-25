package compliancemanager

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/connection"
)

type managerImpl struct {
	sensorConnMgr connection.Manager
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(sensorConnMgr connection.Manager) Manager {
	return &managerImpl{
		sensorConnMgr: sensorConnMgr,
	}
}

func (m *managerImpl) Sync(_ context.Context) {
	//TODO: Sync scan configurations with sensor
}

func (m *managerImpl) ProcessScanRequest(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Upsert to database
	// 2. Push request to Sensor
	panic("implement me")
}

func (m *managerImpl) ProcessRescanRequest(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Validate config exists in database
	// 2. Push request to Sensor
	panic("implement me")
}

func (m *managerImpl) DeleteScan(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Validate config exists in database
	// 2. Push request to Sensor
	panic("implement me")
}
