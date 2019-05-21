package service

import (
	"github.com/stackrox/rox/central/clientca/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

// New returns a new instance of the Service
func New(manager manager.ClientCAManager) grpc.APIService {
	return newService(manager)
}
