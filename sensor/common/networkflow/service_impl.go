package networkflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/internalapi/data/common"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"google.golang.org/grpc"
)

type hostConnections struct {
	connections        map[connection]time.Time
	lastKnownTimestamp time.Time

	mutex sync.Mutex
}

type serviceImpl struct {
	connectionsByHost      map[string]*hostConnections
	connectionsByHostMutex sync.Mutex
}

// connection is an instance of a connection as reported by collector
type connection struct {
	srcAddr     string
	dstAddr     string
	dstPort     uint16
	containerID string
	protocol    data.L4Protocol
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterNetworkConnectionInfoServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// There is no grpc gateway handler for network connection info service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushNetworkConnectionInfo(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) receiveMessages(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	var hostname string

	msg, err := stream.Recv()
	if err != nil {
		log.Errorf("error dequeueing message: %s", err)
		return err
	}

	registerReq := msg.GetRegister()
	if registerReq == nil {
		return errors.New("unexpected message: expected a register message")
	}

	hostname = registerReq.GetHostname()
	hostConnections := s.registerCollector(hostname)

	isFirst := true
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Errorf("error dequeueing message: %s", err)
			return err
		}

		networkInfoMsg := msg.GetInfo()
		networkInfoMsgTimestamp := time.Now()

		if networkInfoMsg != nil {
			s.processNetworkInfo(hostConnections, networkInfoMsg, networkInfoMsgTimestamp, isFirst)
			isFirst = false
		} else {
			return fmt.Errorf("received unexpected message type %T from hostname %s", networkInfoMsg, hostname)
		}
	}
}

func (s *serviceImpl) registerCollector(hostname string) *hostConnections {

	s.connectionsByHostMutex.Lock()
	defer s.connectionsByHostMutex.Unlock()

	conns := s.connectionsByHost[hostname]
	if conns == nil {
		conns = &hostConnections{
			connections: make(map[connection]time.Time),
		}
	}

	conns.lastKnownTimestamp = time.Now()
	return conns
}

func (s *serviceImpl) processNetworkInfo(existingConnections *hostConnections, networkInfo *sensor.NetworkConnectionInfo, currTimestamp time.Time, isFirst bool) {
	updatedConnections := getUpdatedConnections(networkInfo)

	existingConnections.mutex.Lock()
	defer existingConnections.mutex.Unlock()

	if isFirst {
		for c := range existingConnections.connections {
			// Mark all connections as closed this is the first update
			// after a connection went down and came back up again.
			existingConnections.connections[c] = existingConnections.lastKnownTimestamp
		}
	}

	for c, t := range updatedConnections {
		// timestamp = zero implies the connection is newly added. Add new connections, update existing ones to mark them closed
		existingConnections.connections[c] = t
	}

	existingConnections.lastKnownTimestamp = currTimestamp
}

func getUpdatedConnections(networkInfo *sensor.NetworkConnectionInfo) map[connection]time.Time {
	updatedConnections := make(map[connection]time.Time)

	for _, conn := range networkInfo.GetUpdatedConnections() {
		// Ignore connection originating from a server
		if conn.Role != data.Role_ROLE_CLIENT {
			continue
		}
		c := connection{
			srcAddr:     string(conn.GetLocalAddress().GetAddressData()),
			dstAddr:     string(conn.GetRemoteAddress().GetAddressData()),
			dstPort:     uint16(conn.GetRemoteAddress().GetPort()),
			containerID: conn.GetContainerId(),
			protocol:    conn.GetProtocol(),
		}

		// timestamp will be set to close timestamp for closed connections, and zero for newly added connection.
		if conn.CloseTimestamp != nil {
			timestamp, err := types.TimestampFromProto(conn.CloseTimestamp)
			if err != nil {
				log.Errorf("Unable to convert close timestamp in proto: %s", conn.CloseTimestamp)
				continue
			}
			updatedConnections[c] = timestamp
		} else {
			updatedConnections[c] = time.Unix(0, 0).UTC()
		}

	}

	return updatedConnections
}
