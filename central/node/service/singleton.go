package service

import (
	"github.com/stackrox/stackrox/central/node/globaldatastore"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	serviceInstance     grpc.APIService
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance for the node service.
func Singleton() grpc.APIService {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(globaldatastore.Singleton())
	})
	return serviceInstance
}
