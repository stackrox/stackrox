package service

import (
	"context"

	"github.com/stackrox/rox/central/views/vmcve"
	componentDS "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore"
	cveDS "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore"
	scanDS "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore"
	vmDS "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the VirtualMachineV2Service.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v2.VirtualMachineV2ServiceServer
}

// New returns a new Service instance using the given datastores and view.
func New(
	vmDataStore vmDS.DataStore,
	cveDataStore cveDS.DataStore,
	componentDataStore componentDS.DataStore,
	scanDataStore scanDS.DataStore,
	cveView vmcve.CveView,
) Service {
	return &serviceImpl{
		vmDS:        vmDataStore,
		cveDS:       cveDataStore,
		componentDS: componentDataStore,
		scanDS:      scanDataStore,
		cveView:     cveView,
	}
}
