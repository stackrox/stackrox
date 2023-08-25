package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	// To access summaries, we require users to have view access to every summarized resource.
	// We could consider allowing people to get summaries for just the things they have access to,
	// but that requires non-trivial refactoring, so we'll do it if we feel the need later.
	// This variable is package-level to facilitate the unit test that asserts
	// that it covers all the summarized categories.
	// The keys are matched to fields in the v1.SummaryCountsResponse struct.
	summaryTypeToResourceMetadata = map[string]permissions.ResourceMetadata{
		"NumAlerts":      resources.Alert,
		"NumClusters":    resources.Cluster,
		"NumNodes":       resources.Node,
		"NumDeployments": resources.Deployment,
		"NumImages":      resources.Image,
		"NumSecrets":     resources.Secret,
	}
)

// SearchService provides APIs for search.
type serviceImpl struct {
	v1.UnimplementedSummaryServiceServer

	alerts      alertDataStore.DataStore
	clusters    clusterDataStore.DataStore
	deployments deploymentDataStore.DataStore
	images      imageDataStore.DataStore
	secrets     secretDataStore.DataStore
	nodes       nodeDataStore.DataStore

	authorizer authz.Authorizer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSummaryServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSummaryServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) initializeAuthorizer() {
	requiredPermissions := make([]permissions.ResourceWithAccess, 0, len(summaryTypeToResourceMetadata))
	for _, resource := range summaryTypeToResourceMetadata {
		requiredPermissions = append(requiredPermissions, permissions.View(resource))
	}
	s.authorizer = perrpc.FromMap(
		map[authz.Authorizer][]string{
			user.With(requiredPermissions...): {
				"/v1.SummaryService/GetSummaryCounts",
			},
		},
	)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, s.authorizer.Authorized(ctx, fullMethodName)
}

// GetSummaryCounts returns the global counts of alerts, clusters, deployments, and images.
func (s *serviceImpl) GetSummaryCounts(ctx context.Context, _ *v1.Empty) (*v1.SummaryCountsResponse, error) {
	alerts, err := s.alerts.CountAlerts(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	numClusters, err := s.clusters.CountClusters(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	numNodes, err := s.nodes.CountNodes(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	deployments, err := s.deployments.CountDeployments(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	images, err := s.images.CountImages(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	secrets, err := s.secrets.CountSecrets(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &v1.SummaryCountsResponse{
		NumAlerts:      int64(alerts),
		NumClusters:    int64(numClusters),
		NumNodes:       int64(numNodes),
		NumDeployments: int64(deployments),
		NumImages:      int64(images),
		NumSecrets:     int64(secrets),
	}, nil
}
