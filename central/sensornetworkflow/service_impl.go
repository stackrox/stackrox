package sensornetworkflow

import (
	"fmt"
	"sync"

	protobuf "github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceImpl struct {
	clusterStore store.ClusterStore

	lastUpdateTSMutex sync.Mutex
	lastUpdateTS      map[string]timestamp.MicroTS
}

type networkFlowProperties struct {
	srcDeploymentID string
	dstDeploymentID string
	dstPort         uint32
	protocol        v1.L4Protocol
}

func fromProto(protoProps *v1.NetworkFlowProperties) networkFlowProperties {
	return networkFlowProperties{
		srcDeploymentID: protoProps.SrcDeploymentId,
		dstDeploymentID: protoProps.DstDeploymentId,
		dstPort:         protoProps.DstPort,
		protocol:        protoProps.L4Protocol,
	}
}

func (n *networkFlowProperties) toProto() *v1.NetworkFlowProperties {
	return &v1.NetworkFlowProperties{
		SrcDeploymentId: n.srcDeploymentID,
		DstDeploymentId: n.dstDeploymentID,
		DstPort:         n.dstPort,
		L4Protocol:      n.protocol,
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	central.RegisterNetworkFlowServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PushNetworkFlows(stream central.NetworkFlowService_PushNetworkFlowsServer) error {
	return s.receiveNetworkFlowUpdates(stream)
}

func (s *serviceImpl) receiveNetworkFlowUpdates(stream central.NetworkFlowService_PushNetworkFlowsServer) error {

	clusterID, err := getClusterID(stream)
	if err != nil {
		return status.Errorf(codes.Internal, "unable to get cluster ID from sensor stream")
	}

	isFirst := true
	for {
		update, err := stream.Recv()
		if err != nil {
			return status.Errorf(codes.Internal, "error dequeing message from stream; %s", err)
		}

		updatedFlows := update.Updated
		if updatedFlows == nil {
			return status.Errorf(codes.Internal, "received empty updated flows")
		}

		err = s.updateFlowStore(clusterID, updatedFlows, update.Time, isFirst)
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
		isFirst = false
	}
}

func (s *serviceImpl) updateFlowStore(clusterID string, newFlows []*v1.NetworkFlow, updateTS *protobuf.Timestamp, isFirst bool) error {
	flowStore := s.clusterStore.GetFlowStore(clusterID)

	tsOffset := timestamp.Now() - timestamp.FromProtobuf(updateTS)
	updatedFlows := make(map[networkFlowProperties]timestamp.MicroTS, len(newFlows))

	if isFirst {
		existingFlows, lastUpdateTS, err := flowStore.GetAllFlows()

		if err != nil {
			return fmt.Errorf("unable to get existing flows for cluster %s from store", clusterID)
		}

		for _, flow := range existingFlows {
			updatedFlows[fromProto(flow.GetProps())] = timestamp.FromGogoProtobuf(lastUpdateTS)

		}
	}

	for _, newFlow := range newFlows {
		t := timestamp.FromProtobuf(newFlow.LastSeenTimestamp)
		if newFlow.LastSeenTimestamp != nil {
			t = t + tsOffset
		}
		updatedFlows[fromProto(newFlow.GetProps())] = t
	}

	flowsToBeUpserted := make([]*v1.NetworkFlow, len(updatedFlows))
	for props, ts := range updatedFlows {
		toBeUpserted := &v1.NetworkFlow{
			Props: props.toProto(),
		}
		if ts == 0 {
			toBeUpserted.LastSeenTimestamp = nil
		} else {
			toBeUpserted.LastSeenTimestamp = ts.GogoProtobuf()
		}

		flowsToBeUpserted = append(flowsToBeUpserted, toBeUpserted)
	}

	return flowStore.UpsertFlows(flowsToBeUpserted, timestamp.Now())
}

func getClusterID(stream central.NetworkFlowService_PushNetworkFlowsServer) (string, error) {
	ctx := stream.Context()

	id, err := authn.FromTLSContext(ctx)
	if err != nil {
		return "", err
	}

	if id.Subject.Identifier == "" {
		return "", fmt.Errorf("cluster ID not provided")
	}

	return id.Subject.Identifier, nil

}
