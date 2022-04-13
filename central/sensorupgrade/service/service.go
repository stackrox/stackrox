package service

import (
	"context"

	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/central/sensorupgradeconfig/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service is the v1.SensorUpgrade service.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SensorUpgradeServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(configDataStore datastore.DataStore, manager connection.Manager) Service {
	return &service{
		configDataStore: configDataStore,
		manager:         manager,
	}
}
