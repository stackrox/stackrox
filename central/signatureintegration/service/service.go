package service

import (
	"context"

	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/central/signatureintegration/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the service for managing signature integrations.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SignatureIntegrationServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(store datastore.DataStore, reprocessingLoop reprocessor.Loop) Service {
	return &serviceImpl{
		datastore:        store,
		reprocessingLoop: reprocessingLoop,
	}
}
