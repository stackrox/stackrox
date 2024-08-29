package collector

import (
	"context"

	// "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	// "github.com/grpc-ecosystem/grpc-gateway/runtime"
        "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	// "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	// "github.com/stackrox/rox/pkg/protocompat"
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
	log.Info("In ProcessMessage")
	if msg.GetClusterConfig() != nil && msg.GetClusterConfig().GetConfig() != nil && msg.GetClusterConfig().GetConfig().GetCollectorConfig() != nil {
		log.Infof("msg= %+v", msg)
		s.collectorC <- common.MessageToCollectorWithAddress{
			Msg: &sensor.MsgToCollector{
				Msg: &sensor.MsgToCollector_CollectorConfig {
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
	log.Infof("In add")

	// c.connectionMap[node] = connection
	c.connectionMap[connection] = true
	log.Infof("len(c.connectionMap)= %+v", len(c.connectionMap))
}

func (c *connectionManager) remove(connection sensor.CollectorService_CommunicateServer) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	log.Info("In remove")
	delete(c.connectionMap, connection)
	log.Infof("len(c.connectionMap)= %+v", len(c.connectionMap))
}

func (s *serviceImpl) startSendingLoop() {
	log.Info("In startSendingLoop")
	for msg := range s.collectorC {
		log.Infof("msg %+v", msg)
		log.Infof("len(s.connectionManager.connectionMap)= %+v", len(s.connectionManager.connectionMap))
		for conn := range s.connectionManager.connectionMap {
			log.Infof("Sending msg")
			err := conn.Send(msg.Msg)
			if err != nil {
				log.Info("Sending msg failed")
				return
			}
		}
	}
}

//func (s *serviceImpl) Communicate(empty *protocompat.Empty, server sensor.CollectorService_CommunicateServer) error {
func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {
	log.Info("In Communicate")

	s.connectionManager.add(server)
	defer s.connectionManager.remove(server)

	go s.startSendingLoop()

	return nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	log.Info("In RegisterServiceServer")
	sensor.RegisterCollectorServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	log.Info("In AuthFuncOverride")
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}
