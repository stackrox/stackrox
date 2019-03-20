package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/pkg/grpc"
)

var (
	serviceInstance     grpc.APIService
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance for the node service.
func Singleton() grpc.APIService {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(globalstore.Singleton())
	})
	return serviceInstance
}
