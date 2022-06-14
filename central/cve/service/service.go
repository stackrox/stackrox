package service

import (
	"context"

	cveDataStore "github.com/stackrox/stackrox/central/cve/image/datastore"
	vulnReqMgr "github.com/stackrox/stackrox/central/vulnerabilityrequest/manager/requestmgr"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
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
