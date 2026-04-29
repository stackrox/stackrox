package service

import (
	"context"

	imagev2DS "github.com/stackrox/rox/central/imagev2/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the v2 image export functionality.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v2.ImageExportServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(imageDS imagev2DS.DataStore) Service {
	return &serviceImpl{
		imageDS: imageDS,
	}
}
