package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkbaseline/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/networkgraph"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.NetworkBaseline)): {
			"/v1.NetworkBaselineService/GetNetworkBaseline",
			"/v1.NetworkBaselineService/GetNetworkBaselineStatusForFlows",
		},
		user.With(permissions.Modify(resources.NetworkBaseline)): {
			"/v1.NetworkBaselineService/ModifyBaselineStatusForPeers",
			"/v1.NetworkBaselineService/LockNetworkBaseline",
			"/v1.NetworkBaselineService/UnlockNetworkBaseline",
		},
	})
)

type serviceImpl struct {
	datastore datastore.ReadOnlyDataStore
	manager   manager.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNetworkBaselineServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNetworkBaselineServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetNetworkBaselineStatusForFlows(
	ctx context.Context,
	request *v1.NetworkBaselineStatusRequest,
) (*v1.NetworkBaselineStatusResponse, error) {
	// Check if the baseline for deployment indeed exists
	baseline, found, err := s.datastore.GetNetworkBaseline(ctx, request.GetDeploymentId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.New("network baseline for the deployment does not exist")
	}

	// Got the baseline, check status of each passed in peer
	statuses := s.getStatusesForPeers(baseline, request.GetPeers())
	return &v1.NetworkBaselineStatusResponse{Statuses: statuses}, nil
}

func (s *serviceImpl) GetNetworkBaseline(
	ctx context.Context,
	request *v1.ResourceByID,
) (*storage.NetworkBaseline, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "Network baseline id must be provided")
	}
	baseline, found, err := s.datastore.GetNetworkBaseline(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errorhelpers.ErrNotFound, "network baseline with id %q does not exist", request.GetId())
	}

	return baseline, nil
}

func (s *serviceImpl) getStatusesForPeers(
	baseline *storage.NetworkBaseline,
	examinedPeers []*v1.NetworkBaselineStatusPeer,
) []*v1.NetworkBaselinePeerStatus {
	baselinePeerByID := s.getBaselinePeerByEntityID(baseline)

	statuses := make([]*v1.NetworkBaselinePeerStatus, 0, len(examinedPeers))
	for _, examinedPeer := range examinedPeers {
		status := v1.NetworkBaselinePeerStatus_ANOMALOUS
		examinedPeerKey := networkgraph.Entity{
			Type: examinedPeer.GetEntity().GetType(),
			ID:   examinedPeer.GetEntity().GetId(),
		}
		if baselinePeer, ok := baselinePeerByID[examinedPeerKey]; ok {
			for _, baselineProperty := range baselinePeer.GetProperties() {
				if examinedPeer.GetProtocol() == baselineProperty.GetProtocol() &&
					examinedPeer.GetPort() == baselineProperty.GetPort() &&
					examinedPeer.GetIngress() == baselineProperty.GetIngress() {
					// Matched with what we have in the baseline
					status = v1.NetworkBaselinePeerStatus_BASELINE
					break
				}
			}
		}
		statuses =
			append(
				statuses,
				&v1.NetworkBaselinePeerStatus{
					Peer:   examinedPeer,
					Status: status,
				})
	}

	return statuses
}

func (s *serviceImpl) getBaselinePeerByEntityID(
	baseline *storage.NetworkBaseline,
) map[networkgraph.Entity]*storage.NetworkBaselinePeer {
	result := make(map[networkgraph.Entity]*storage.NetworkBaselinePeer, len(baseline.GetPeers()))

	peers := baseline.GetPeers()
	for _, peer := range peers {
		key := networkgraph.Entity{
			Type: peer.GetEntity().GetInfo().GetType(),
			ID:   peer.GetEntity().GetInfo().GetId(),
		}
		result[key] = peer
	}

	return result
}

func (s *serviceImpl) ModifyBaselineStatusForPeers(ctx context.Context, request *v1.ModifyBaselineStatusForPeersRequest) (*v1.Empty, error) {
	err := s.manager.ProcessBaselineStatusUpdate(ctx, request)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) LockNetworkBaseline(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	err := s.manager.ProcessBaselineLockUpdate(ctx, request.Id, true)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UnlockNetworkBaseline(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	err := s.manager.ProcessBaselineLockUpdate(ctx, request.Id, false)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
