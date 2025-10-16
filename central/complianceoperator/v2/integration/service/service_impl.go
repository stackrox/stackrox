package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000

	fmtGetClusterErr       = "Unable to get information for cluster %q."
	fmtGetClusterNotFound  = "Cluster %q not found."
	fmtGetClusterUnhealthy = "Sensor connection to cluster %q is not healthy."
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance), permissions.View(resources.Cluster)): {
			v2.ComplianceIntegrationService_ListComplianceIntegrations_FullMethodName,
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceMetaDataStore complianceDS.DataStore, clusterStore datastore.DataStore) Service {
	return &serviceImpl{
		complianceMetaDataStore: complianceMetaDataStore,
		clusterDS:               clusterStore,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceIntegrationServiceServer

	complianceMetaDataStore complianceDS.DataStore
	clusterDS               datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) ListComplianceIntegrations(ctx context.Context, req *v2.RawQuery) (*v2.ListComplianceIntegrationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, req.GetPagination(), maxPaginationLimit)

	integrations, err := s.complianceMetaDataStore.GetComplianceIntegrationsView(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve compliance integrations.")
	}

	apiIntegrations, err := convertStorageProtos(ctx, integrations, s.complianceMetaDataStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert compliance integrations.")
	}

	clusterSensorStatuses := map[string]storage.ClusterHealthStatus_HealthStatusLabel{}
	errFetchStatus := s.clusterDS.WalkClusters(ctx, func(cluster *storage.Cluster) error {
		clusterSensorStatuses[cluster.GetId()] = cluster.GetHealthStatus().GetSensorHealthStatus()
		return nil
	})

	// Enrich cluster status with sensor connection status.
	for _, apiIntegration := range apiIntegrations {
		if errFetchStatus != nil {
			apiIntegration.SetStatusErrors(append(apiIntegration.GetStatusErrors(), fmt.Sprintf(fmtGetClusterErr, apiIntegration.GetClusterName())))
			continue
		}
		clusterSensorStatus, found := clusterSensorStatuses[apiIntegration.GetClusterId()]
		if !found {
			apiIntegration.SetStatusErrors(append(apiIntegration.GetStatusErrors(), fmt.Sprintf(fmtGetClusterNotFound, apiIntegration.GetClusterName())))
			continue
		}
		if clusterSensorStatus != storage.ClusterHealthStatus_HEALTHY {
			apiIntegration.SetStatusErrors(append(apiIntegration.GetStatusErrors(), fmt.Sprintf(fmtGetClusterUnhealthy, apiIntegration.GetClusterName())))
		}
	}

	integrationCount, err := s.complianceMetaDataStore.CountIntegrations(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine number of compliance operator integrations")
	}

	lcir := &v2.ListComplianceIntegrationsResponse{}
	lcir.SetIntegrations(apiIntegrations)
	lcir.SetTotalCount(int32(integrationCount))
	return lcir, nil
}
