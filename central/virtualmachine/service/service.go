package service

import (
	"context"

	"github.com/stackrox/rox/central/administration/events"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/virtualmachine/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule(events.EnableAdministrationEvents())
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.VirtualMachineServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	datastore datastore.DataStore,
	clusterSACHelper sachelper.ClusterSacHelper,
) Service {
	return &serviceImpl{
		datastore:        datastore,
		clusterSACHelper: clusterSACHelper,
	}
}
