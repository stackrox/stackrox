package service

import (
	"github.com/stackrox/rox/central/license/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// New creates a new license service
func New(licenseStatus *v1.Metadata_LicenseStatus, licenseMgr manager.LicenseManager) grpc.APIService {
	return newService(licenseStatus, licenseMgr)
}
