package service

import (
	"context"

	cveDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	vulnReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves cve data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ImageCVEServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	cveDataStore cveDataStore.DataStore,
	vulnReqMgr vulnReqMgr.Manager,
) Service {
	return &serviceImpl{
		cves:       cveDataStore,
		vulnReqMgr: vulnReqMgr,
	}
}
