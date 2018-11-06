package service

import (
	"sync"

	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/pkg/grpc"
)

var (
	serviceInstance     grpc.APIService
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance for the node service.
func Singleton() grpc.APIService {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(store.Singleton())
	})
	return serviceInstance
}
