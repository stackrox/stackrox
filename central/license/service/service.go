package service

import (
	"github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/pkg/grpc"
)

// New creates a new license service
func New(lockdownMode bool, licenseMgr manager.LicenseManager) grpc.APIService {
	return newService(lockdownMode, licenseMgr)
}
