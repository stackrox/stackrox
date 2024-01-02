package service

import (
	"context"

	"github.com/stackrox/rox/central/teams/datastore"
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
}

// Singleton provides the service for teams.
func Singleton() Service {
	once.Do(func() {
		s = &serviceImpl{
			ds: datastore.Singleton(),
		}
	})
	return s
}
