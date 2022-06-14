package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/node/globaldatastore"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/errox"
	pkgGRPC "github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Node)): {
			"/v1.NodeService/GetNode",
			"/v1.NodeService/ListNodes",
		},
	})
)

type nodeServiceImpl struct {
	nodeStore globaldatastore.GlobalDataStore
}

// New creates a new node service from the given node store.
func New(nodeStore globaldatastore.GlobalDataStore) pkgGRPC.APIService {
	return &nodeServiceImpl{
		nodeStore: nodeStore,
	}
}

func (s *nodeServiceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterNodeServiceServer(server, s)
}

func (s *nodeServiceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNodeServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *nodeServiceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *nodeServiceImpl) ListNodes(ctx context.Context, req *v1.ListNodesRequest) (*v1.ListNodesResponse, error) {
	clusterLocalStore, err := s.nodeStore.GetClusterNodeStore(ctx, req.GetClusterId(), false)
	if err != nil {
		return nil, errors.Errorf("could not access per-cluster node store for cluster %q: %v", req.GetClusterId(), err)
	}

	nodes, err := clusterLocalStore.ListNodes()
	if err != nil {
		return nil, errors.Errorf("could not list notes in cluster %s: %v", req.GetClusterId(), err)
	}
	return &v1.ListNodesResponse{
		Nodes: nodes,
	}, nil
}

func (s *nodeServiceImpl) GetNode(ctx context.Context, req *v1.GetNodeRequest) (*storage.Node, error) {
	clusterLocalStore, err := s.nodeStore.GetClusterNodeStore(ctx, req.GetClusterId(), false)
	if err != nil {
		return nil, errors.Errorf("could not access per-cluster node store for cluster %q: %v", req.GetClusterId(), err)
	}

	node, err := clusterLocalStore.GetNode(req.GetNodeId())
	if err != nil {
		return nil, errors.Errorf("could not locate node %q in per-cluster node store for cluster %s: %v", req.GetNodeId(), req.GetClusterId(), err)
	}

	if node == nil {
		return nil, errors.Wrapf(errox.NotFound, "node %q in cluster %q does not exist", req.GetNodeId(), req.GetClusterId())
	}

	return node, nil
}
