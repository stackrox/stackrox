package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	GetAuthStatus(ctx context.Context, request *empty.Empty) (*v1.AuthStatus, error)
}

// New returns a new Service instance using the given DataStore.
func New() Service {
	return &serviceImpl{}
}
