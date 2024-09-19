package collector

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// CollectorService is the struct that manages the collector configuration
type serviceImpl struct {
	sensor.UnimplementedCollectorServiceServer

	collectorC chan common.MessageToCollectorWithAddress

	connectionManager *connectionManager
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
}

func (s *serviceImpl) Start() error {
	s.collectorC = make(chan common.MessageToCollectorWithAddress)
	return nil
}

func (s *serviceImpl) Stop(_ error) {}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (s *serviceImpl) ProcessMessage(msg *central.MsgToSensor) error {
	if msg.GetClusterConfig() != nil && msg.GetClusterConfig().GetConfig() != nil && msg.GetClusterConfig().GetConfig().GetCollectorConfig() != nil {
		s.collectorC <- common.MessageToCollectorWithAddress{
			Msg: &sensor.MsgToCollector{
				Msg: &sensor.MsgToCollector_CollectorConfig{
					CollectorConfig: msg.GetClusterConfig().GetConfig().GetCollectorConfig(),
				},
			},
			Broadcast: true,
		}
	}

	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

type connectionManager struct {
	connectionLock sync.RWMutex
	connectionMap  map[sensor.CollectorService_CommunicateServer]bool
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connectionMap: make(map[sensor.CollectorService_CommunicateServer]bool),
	}
}

func (c *connectionManager) add(connection sensor.CollectorService_CommunicateServer) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	c.connectionMap[connection] = true
}

func (c *connectionManager) remove(connection sensor.CollectorService_CommunicateServer) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	delete(c.connectionMap, connection)
}

func (s *serviceImpl) startSendingLoop() {
	for msg := range s.collectorC {
		for conn := range s.connectionManager.connectionMap {
			err := conn.Send(msg.Msg)
			if err != nil {
				log.Info("Sending msg failed")
				return
			}
		}
	}
}

func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {

	s.connectionManager.add(server)
	defer s.connectionManager.remove(server)

	go s.startSendingLoop()

	return nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterCollectorServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}
