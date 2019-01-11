package service

import (
	"github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

// NewService creates and returns a new compliance management service.
func NewService(manager manager.ComplianceManager) grpc.APIService {
	return newService(manager)
}
