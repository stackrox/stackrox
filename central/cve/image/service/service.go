package service

import (
	"context"

	imageCVEV2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageComponentV2DS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	"github.com/stackrox/rox/central/views/imagecve"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service serves aggregated image CVE list/count over REST/gRPC.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ImageCVEServiceServer
}

// New returns a Service wrapping the same ImageCVE view and stores GraphQL uses.
func New(
	cveView imagecve.CveView,
	imageCVEs imageCVEV2DS.DataStore,
	components imageComponentV2DS.DataStore,
	vulnReqs vulnReqDataStore.DataStore,
) Service {
	return &serviceImpl{
		cveView:      cveView,
		imageCVEs:    imageCVEs,
		components:   components,
		vulnReqStore: vulnReqs,
	}
}
