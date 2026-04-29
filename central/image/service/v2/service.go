package service

import (
	"context"

	imagecvev2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the v2 image export functionality.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v2.ImageExportServiceServer
}

// New returns a new Service instance using the given datastores.
func New(images imageDS.DataStore, cves imagecvev2DS.DataStore) Service {
	return &serviceImpl{
		imageDS: images,
		cveDS:   cves,
	}
}
