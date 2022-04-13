package service

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service is the interface for the sensor connection service.
type Service interface {
	grpc.APIService
	central.SensorServiceServer
}
