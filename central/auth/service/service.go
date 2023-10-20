package service

import (
	"context"

	"github.com/stackrox/rox/central/auth/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Service
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	GetAuthStatus(ctx context.Context, request *v1.Empty) (*v1.AuthStatus, error)
}

// Singleton returns a new auth service instance.
func Singleton() Service {
	once.Do(func() {
		svc := &serviceImpl{}
		if features.AuthMachineToMachine.Enabled() {
			svc.authDataStore = datastore.Singleton()
		}
		s = svc
	})
	return s
}
