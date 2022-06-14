package service

import (
	"context"

	cveDataStore "github.com/stackrox/stackrox/central/cve/node/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the microservice that serves cve data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.NodeCVEServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(cveDataStore cveDataStore.DataStore) Service {
	return &serviceImpl{
		cves: cveDataStore,
	}
}
