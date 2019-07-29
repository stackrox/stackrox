package service

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

// New creates a new DB service.
func New(mgr manager.Manager) grpc.APIService {
	return newService(mgr)
}
