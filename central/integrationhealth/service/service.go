package service

import (
	"context"

	"github.com/stackrox/stackrox/central/integrationhealth/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/scanners"
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

// New returns a new Service instance using the given DataStore.
func New(integrationHealthDS datastore.DataStore, vulnDefsInfoProvider scanners.VulnDefsInfoProvider) Service {
	return &serviceImpl{
		datastore:            integrationHealthDS,
		vulnDefsInfoProvider: vulnDefsInfoProvider,
	}
}
