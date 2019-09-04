package service

import (
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the v1.SensorUpgrade service.
type Service interface {
	grpc.APIService

	grpc_auth.ServiceAuthFuncOverride

	central.SensorUpgradeControlServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(connectionManager connection.Manager) Service {
	return &service{
		connectionManager: connectionManager,
	}
}
