package service

import (
	"net/http"

	"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service is the interface for the v2 db backup/restore service. It provides both gRPC as well as HTTP/1.1 service
// handlers.
type Service interface {
	grpc.APIService

	v1.DBServiceServer

	RestoreHandler() http.Handler
	ResumeRestoreHandler() http.Handler
}

// New creates a new DB service.
func New(mgr manager.Manager) Service {
	return newService(mgr)
}
