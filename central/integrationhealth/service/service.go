package service

import (
	"context"

	"github.com/stackrox/rox/central/integrationhealth/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/scanners"
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
