package service

import (
	"context"

	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the public API for risk ranking adjustments.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance using the provided manager.
func New(mgr manager.Manager) Service {
	return &serviceImpl{
		manager: mgr,
	}
}
