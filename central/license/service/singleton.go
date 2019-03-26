package service

import (
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance     grpc.APIService
	instanceInit sync.Once
)

// Singleton returns the singleton instance of the license service.
func Singleton() grpc.APIService {
	instanceInit.Do(func() {
		instance = New()
	})
	return instance
}
