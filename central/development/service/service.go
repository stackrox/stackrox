package service

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the interface for the development connection service.
type Service interface {
	grpc.APIService
	central.DevelopmentServiceServer
}
