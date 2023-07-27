package service

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc"
)

// var (
// 	log = logging.LoggerForModule()
// )

// Service is the GRPC service interface that proves the entry point for serving the cluster target version.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new instance of service.
func New() Service {
	return &serviceImpl{}
}
