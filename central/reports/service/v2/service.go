package v2

import (
	"context"

	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for reports.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	apiV2.ReportServiceServer
}

// New returns a new instance of the service.
func New(metadataDatastore metadataDS.DataStore) Service {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	return &serviceImpl{
		metadataDatastore: metadataDatastore,
	}
}
