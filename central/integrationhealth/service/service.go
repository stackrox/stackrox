package service

import (
	"context"

	"github.com/stackrox/rox/central/integrationhealth/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves integration health data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.IntegrationHealthServiceServer
}

type vulnDefsInfoProvider interface {
	GetVulnDefsInfo() (*v1.VulnDefinitionsInfo, error)
}

// New returns a new Service instance using the given DataStore.
func New(integrationHealthDS datastore.DataStore, vulnDefsInfoProvider vulnDefsInfoProvider) Service {
	return &serviceImpl{
		datastore:            integrationHealthDS,
		vulnDefsInfoProvider: vulnDefsInfoProvider,
	}
}
