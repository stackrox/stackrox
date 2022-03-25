package service

import (
	"context"

	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	vulnReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
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

	v1.CVEServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	cveDataStore cveDataStore.DataStore,
	indexQ queue.WaitableQueue,
	vulnReqMgr vulnReqMgr.Manager,
) Service {
	return &serviceImpl{
		cves:       cveDataStore,
		indexQ:     indexQ,
		vulnReqMgr: vulnReqMgr,
	}
}
