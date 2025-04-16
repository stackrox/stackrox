package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	deploymentUtils "github.com/stackrox/rox/central/deployment/utils"
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkbaseline/manager"
	flowDatastore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

var (
	defaultSince = -1 * time.Hour
	authorizer   = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			v1.NetworkBaselineService_GetNetworkBaseline_FullMethodName,
			v1.NetworkBaselineService_GetNetworkBaselineStatusForFlows_FullMethodName,
			v1.NetworkBaselineService_GetNetworkBaselineStatusForExternalFlows_FullMethodName,
		},
		user.With(permissions.Modify(resources.DeploymentExtension)): {
			v1.NetworkBaselineService_ModifyBaselineStatusForPeers_FullMethodName,
			v1.NetworkBaselineService_LockNetworkBaseline_FullMethodName,
			v1.NetworkBaselineService_UnlockNetworkBaseline_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedNetworkBaselineServiceServer

	datastore datastore.ReadOnlyDataStore
	manager   manager.Manager

	flowStore flowDatastore.ClusterDataStore
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

// GetNetworkBaselineStatusForFlows - gets the status of the flows within the baseline.
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
		baseline, err = s.createBaseline(ctx, request.GetDeploymentId())
		if err != nil {
			return nil, err
		}
	}

	// Got the baseline, check status of each passed in peer
	statuses := s.getStatusesForPeers(baseline, request.GetPeers())
	return &v1.NetworkBaselineStatusResponse{Statuses: statuses}, nil
}

func (s *serviceImpl) GetNetworkBaselineStatusForExternalFlows(ctx context.Context, request *v1.NetworkBaselineExternalStatusRequest) (*v1.NetworkBaselineExternalStatusResponse, error) {
	baseline, found, err := s.datastore.GetNetworkBaseline(ctx, request.GetDeploymentId())
	if err != nil {
		return nil, err
	}

	if !found {
		baseline, err = s.createBaseline(ctx, request.GetDeploymentId())
		if err != nil {
			return nil, err
		}
	}

	since := protocompat.ConvertTimestampToTimeOrNil(request.GetSince())
	if since == nil {
		t := time.Now().Add(defaultSince)
		since = &t
	}

	peers, err := s.manager.GetExternalNetworkPeers(ctx, request.GetDeploymentId(), request.GetQuery(), since)
	if err != nil {
		return nil, err
	}

	statuses := s.getStatusesForPeers(baseline, peers)

	anomalousFlows := make([]*v1.NetworkBaselinePeerStatus, 0)
	baselineFlows := make([]*v1.NetworkBaselinePeerStatus, 0)

	for _, status := range statuses {
		switch status.GetStatus() {
		case v1.NetworkBaselinePeerStatus_ANOMALOUS:
			anomalousFlows = append(anomalousFlows, status)
		case v1.NetworkBaselinePeerStatus_BASELINE:
			baselineFlows = append(baselineFlows, status)
		}
	}

	totalAnomalous := len(anomalousFlows)
	totalBaseline := len(baselineFlows)

	pg := request.GetPagination()
	if pg != nil {
		anomalousFlows = paginated.PaginateSlice(int(pg.Offset), int(pg.Limit), anomalousFlows)
		baselineFlows = paginated.PaginateSlice(int(pg.Offset), int(pg.Limit), baselineFlows)
	}

	return &v1.NetworkBaselineExternalStatusResponse{
		Anomalous:      anomalousFlows,
		TotalAnomalous: int32(totalAnomalous),
		Baseline:       baselineFlows,
		TotalBaseline:  int32(totalBaseline),
	}, nil
}

// GetNetworkBaseline gets the network baseline associated with the deployment.
func (s *serviceImpl) GetNetworkBaseline(
	ctx context.Context,
	request *v1.ResourceByID,
) (*storage.NetworkBaseline, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Deployment id for the network baseline must be provided")
	}
	baseline, found, err := s.datastore.GetNetworkBaseline(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		baseline, err = s.createBaseline(ctx, request.GetId())
		if err != nil {
			return nil, err
		}
	}

	return baseline, nil
}

func (s *serviceImpl) createBaseline(ctx context.Context, deploymentID string) (*storage.NetworkBaseline, error) {
	// We didn't find one but user asked for one.  Let's try to build one
	err := s.manager.CreateNetworkBaseline(deploymentID)
	if err != nil {
		return nil, err
	}

	// Grab the newly created baseline
	baseline, found, err := s.datastore.GetNetworkBaseline(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Network baseline for deployment id %q does not exist", deploymentID)
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

		examinedPeerKey := s.anonymizedPeerKey(examinedPeer)

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

// anonymizedPeerKey anonymizes discovered external peers to the internet
// for the purposes of looking up matching baseline peers.
func (s *serviceImpl) anonymizedPeerKey(peer *v1.NetworkBaselineStatusPeer) networkgraph.Entity {
	entity := peer.GetEntity()
	if entity.GetType() == storage.NetworkEntityInfo_EXTERNAL_SOURCE && entity.GetDiscovered() {
		internet := networkgraph.InternetEntity()
		// removing some fields for map keying
		return networkgraph.Entity{
			Type: internet.Type,
			ID:   internet.ID,
		}
	}

	return networkgraph.Entity{
		Type: entity.GetType(),
		ID:   entity.GetId(),
	}
}

// getBaselinePeerByEntityID indexes the peers from the provided baseline
// by their (type, ID) information.
func (s *serviceImpl) getBaselinePeerByEntityID(
	baseline *storage.NetworkBaseline,
) map[networkgraph.Entity]*storage.NetworkBaselinePeer {
	result := make(map[networkgraph.Entity]*storage.NetworkBaselinePeer, len(baseline.GetPeers()))

	peers := baseline.GetPeers()
	for _, peer := range peers {
		peerType := peer.GetEntity().GetInfo().GetType()
		peerId := peer.GetEntity().GetInfo().GetId()
		key := networkgraph.Entity{
			Type: peerType,
			ID:   peerId,
		}
		result[key] = peer
		// In UI flows, the peers for flow comparison to the baseline are
		// the ones received from the network graph call.
		// Scoped Access Control masking in network graph generates new
		// identifiers for entities that are not in the allowed scope of
		// the requested, and this in a deterministic way.
		// Here, the peer is also referenced by the ID that would be
		// generated for the network graph, so that flows coming from or
		// targeting masked entities would still be flagged as belonging to
		// the network baseline.
		if peerType == storage.NetworkEntityInfo_DEPLOYMENT {
			deploymentName := peer.GetEntity().GetInfo().GetDeployment().GetName()
			maskedKey := networkgraph.Entity{
				Type: peerType,
				ID:   deploymentUtils.GetMaskedDeploymentID(peerId, deploymentName),
			}
			result[maskedKey] = peer
		}
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
