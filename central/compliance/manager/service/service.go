package service

import (
	"github.com/stackrox/stackrox/central/compliance/manager"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// ComplianceManagementService is the RPC service for compliance management.
type ComplianceManagementService interface {
	grpc.APIService
	v1.ComplianceManagementServiceServer
}

// NewService creates and returns a new compliance management service.
func NewService(manager manager.ComplianceManager) ComplianceManagementService {
	return newService(manager)
}
