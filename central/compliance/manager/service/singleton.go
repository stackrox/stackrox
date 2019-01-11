package service

import (
	"sync"

	"github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

var (
	serviceInstance grpc.APIService
	serviceInit     sync.Once
)

// Singleton returns the compliance management service singleton instance.
func Singleton() grpc.APIService {
	serviceInit.Do(func() {
		serviceInstance = NewService(manager.Singleton())
	})
	return serviceInstance
}
