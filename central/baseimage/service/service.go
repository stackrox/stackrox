package service

import (
	"context"

	"github.com/stackrox/rox/central/baseimage/datastore/repository"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves base image data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v2.BaseImageServiceV2Server
}

// New returns a new Service instance using the given DataStore.
func New(datastore repository.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}
