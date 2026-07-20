package service

import (
	"context"

	"github.com/stackrox/rox/central/lightspeed/store"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

// Service provides the interface for the Lightspeed configuration API.
type Service interface {
	pkgGRPC.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.LightspeedServiceServer
}

func initialize() {
	instance = &serviceImpl{
		store:   store.Singleton(),
		connMgr: connection.ManagerSingleton(),
	}
}

// Singleton returns the singleton instance of the Lightspeed service.
func Singleton() Service {
	once.Do(initialize)
	return instance
}
