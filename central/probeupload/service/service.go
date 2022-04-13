package service

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service is the service for collector probe upload handling.
type Service interface {
	grpc.APIServiceWithCustomRoutes

	v1.ProbeUploadServiceServer
}
