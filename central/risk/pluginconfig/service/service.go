package service

import (
	"context"

	"github.com/stackrox/rox/central/risk/scorer/plugin/registry"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for risk scoring plugin configurations.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.RiskScoringPluginServiceServer
}

// New returns a new instance of the service.
func New(reg registry.Registry) Service {
	return &serviceImpl{
		registry: reg,
	}
}
