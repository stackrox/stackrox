package service

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Service
)

// Service provides the interface.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	GetAuthStatus(ctx context.Context, request *v1.Empty) (*v1.AuthStatus, error)
}

func Singleton() Service {
	once.Do(func() {

	})
	return s
}
