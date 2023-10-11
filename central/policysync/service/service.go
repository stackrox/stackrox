package service

import (
	"context"

	"github.com/stackrox/rox/central/policysync/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Service
)

// Service provides the service.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.PolicySyncServiceServer
}

// Singleton provides the singleton instance.
func Singleton() Service {
	once.Do(func() {
		s = newService(datastore.Singleton())
	})
	return s
}

func newService(ds datastore.DataStore) Service {
	return &serviceImpl{
		ds: ds,
	}
}
