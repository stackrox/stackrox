package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
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
	v1.UnimplementedNodeServiceServer

	nodeDatastore datastore.DataStore
}

// New creates a new node service from the given node store.
func New(nodeDatastore datastore.DataStore) pkgGRPC.APIService {
	return &nodeServiceImpl{
		nodeDatastore: nodeDatastore,
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
	nodes, err := s.nodeDatastore.SearchRawNodes(ctx,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, req.GetClusterId()).ProtoQuery())
	if err != nil {
		return nil, errors.Errorf("could not list notes in cluster %s: %v", req.GetClusterId(), err)
	}
	return &v1.ListNodesResponse{
		Nodes: nodes,
	}, nil
}

func (s *nodeServiceImpl) GetNode(ctx context.Context, req *v1.GetNodeRequest) (*storage.Node, error) {
	// Previously, the cluster ID in the request was used to obtain global store. However, when the datastore interfaces
	// were consolidated, the global store was dropped. Nodes have non-composite unique node ID across all clusters.
	// Hence, Cluster ID is not required to retrieve a node.
	//
	// The GRPC endpoint (that includes the request format) was not changed to avoid any user-facing changes (/deprecation).
	node, found, err := s.nodeDatastore.GetNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, errors.Errorf("could not locate node %q for cluster %s: %v", req.GetNodeId(), req.GetClusterId(), err)
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "node %q in cluster %q does not exist", req.GetNodeId(), req.GetClusterId())
	}
	return node, nil
}
