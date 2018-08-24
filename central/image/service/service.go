package service

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetImage(ctx context.Context, request *v1.ResourceByID) (*v1.Image, error)
	ListImages(ctx context.Context, request *v1.RawQuery) (*v1.ListImagesResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}
