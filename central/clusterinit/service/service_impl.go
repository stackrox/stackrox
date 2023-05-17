package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusterinit/backend"
	"github.com/stackrox/rox/central/clusterinit/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = user.WithRole(accesscontrol.Admin)
)

var _ v1.ClusterInitServiceServer = (*serviceImpl)(nil)

type serviceImpl struct {
	v1.UnimplementedClusterInitServiceServer

	backend      backend.Backend
	clusterStore clusterStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterClusterInitServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterClusterInitServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetInitBundles(ctx context.Context, _ *v1.Empty) (*v1.InitBundleMetasResponse, error) {
	initBundleMetas, err := s.backend.GetAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving meta data for all init bundles")
	}
	bundlesIDs := set.NewStringSet()
	for _, b := range initBundleMetas {
		bundlesIDs.Add(b.GetId())
	}
	impactedClustersForBundles, err := s.getImpactedClustersForBundles(ctx, bundlesIDs)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving clusters for all init bundles")
	}

	v1InitBundleMetas := make([]*v1.InitBundleMeta, 0, len(initBundleMetas))
	for _, initBundle := range initBundleMetas {
		v1InitBundleMetas = append(v1InitBundleMetas,
			initBundleMetaStorageToV1WithImpactedClusters(initBundle, impactedClustersForBundles[initBundle.GetId()]))
	}

	return &v1.InitBundleMetasResponse{Items: v1InitBundleMetas}, nil
}

func (s *serviceImpl) GetCAConfig(ctx context.Context, _ *v1.Empty) (*v1.GetCAConfigResponse, error) {
	caConfig, err := s.backend.GetCAConfig(ctx)
	if err != nil {
		return nil, errors.New("retrieving meta data for all ")
	}

	caConfigYAML, err := caConfig.RenderAsYAML()
	if err != nil {
		return nil, errors.Errorf("failed to render CA config to YAML: %v", err)
	}

	return &v1.GetCAConfigResponse{
		HelmValuesBundle: caConfigYAML,
	}, nil
}

func (s *serviceImpl) GenerateInitBundle(ctx context.Context, request *v1.InitBundleGenRequest) (*v1.InitBundleGenResponse, error) {
	generated, err := s.backend.Issue(ctx, request.GetName())
	if err != nil {
		if errors.Is(err, store.ErrInitBundleDuplicateName) {
			return nil, status.Errorf(codes.AlreadyExists, "generating new init bundle: %s", err)
		}
		return nil, errors.Errorf("generating new init bundle: %s", err)
	}
	meta := initBundleMetaStorageToV1(generated.Meta)

	bundleYaml, err := generated.RenderAsYAML()
	if err != nil {
		return nil, errors.Errorf("rendering init bundle as YAML: %s", err)
	}
	bundleK8sManifest, err := generated.RenderAsK8sSecrets()
	if err != nil {
		return nil, errors.Errorf("rendering init bundle as Kubernetes secrets: %s", err)
	}

	return &v1.InitBundleGenResponse{
		HelmValuesBundle: bundleYaml,
		KubectlBundle:    bundleK8sManifest,
		Meta:             meta,
	}, nil
}

func (s *serviceImpl) RevokeInitBundle(ctx context.Context, request *v1.InitBundleRevokeRequest) (*v1.InitBundleRevokeResponse, error) {
	var failed []*v1.InitBundleRevokeResponse_InitBundleRevocationError
	var revoked []string

	userConfirmedImpactedClusters := request.GetConfirmImpactedClustersIds()
	impactedClustersForBundles, err := s.getImpactedClustersForBundles(ctx, set.NewStringSet(request.GetIds()...))
	if err != nil {
		return nil, err
	}

	for _, id := range request.GetIds() {
		impactedClusters := impactedClustersForBundles[id]
		if !containsAll(userConfirmedImpactedClusters, impactedClusters) {
			failed = append(failed, &v1.InitBundleRevokeResponse_InitBundleRevocationError{
				Id:               id,
				Error:            "not all clusters were confirmed",
				ImpactedClusters: impactedClusters,
			})
		} else if err := s.backend.Revoke(ctx, id); err != nil {
			failed = append(failed, &v1.InitBundleRevokeResponse_InitBundleRevocationError{Id: id, Error: err.Error()})
		} else {
			revoked = append(revoked, id)
		}
	}

	return &v1.InitBundleRevokeResponse{InitBundleRevokedIds: revoked, InitBundleRevocationErrors: failed}, nil
}

func (s *serviceImpl) getImpactedClustersForBundles(ctx context.Context, bundleIDs set.StringSet) (map[string][]*v1.InitBundleMeta_ImpactedCluster, error) {
	clusters, err := s.clusterStore.GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not list clusters")
	}
	clustersByBundleID := make(map[string][]*v1.InitBundleMeta_ImpactedCluster, len(bundleIDs))
	for _, cluster := range clusters {
		bundleID := cluster.GetInitBundleId()
		if bundleIDs.Contains(bundleID) {
			clustersByBundleID[bundleID] = append(clustersByBundleID[bundleID], &v1.InitBundleMeta_ImpactedCluster{
				Name: cluster.GetName(),
				Id:   cluster.GetId(),
			})
		}
	}
	return clustersByBundleID, nil
}

func containsAll(clusterIDs []string, clusters []*v1.InitBundleMeta_ImpactedCluster) bool {
	ids := set.NewStringSet(clusterIDs...)
	for _, c := range clusters {
		if !ids.Contains(c.GetId()) {
			return false
		}
	}
	return true
}
