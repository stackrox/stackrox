package service

import (
	"context"

	"github.com/stackrox/rox/central/debugactions/manager"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves deployment data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v2.DebugActionServiceServer
}

// New returns a new service instance
func New(actionMgr manager.Manager) Service {
	return &serviceImpl{
		actionMgr: actionMgr,
	}
}
