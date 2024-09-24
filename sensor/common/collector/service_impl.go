package collector

import (
	"context"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	log = logging.LoggerForModule()
)

const (
	// CollectorHelloMetadataKey is the key to indicate both sensor and collector that collector, not sensor,
	// will be the first to send a message on the stream. Sensor must not send keys
	// unless it received header metadata from collector.
	CollectorHelloMetadataKey = "Rox-Collector-Hello"
)

// CollectorService is the struct that manages the collector configuration
type serviceImpl struct {
	sensor.UnimplementedCollectorServiceServer

	collectorC chan *sensor.MsgToCollector

	connectionManager *connectionManager
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
}

func (s *serviceImpl) Start() error {
	s.collectorC = make(chan *sensor.MsgToCollector)
	return nil
}

func (s *serviceImpl) Stop(_ error) {
	close(s.collectorC)
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func getCollectorConfig(msg *central.MsgToSensor) *storage.CollectorConfig {
	if clusterConfig := msg.GetClusterConfig(); clusterConfig != nil {
		if config := clusterConfig.GetConfig(); config != nil {
			if collectorConfig := config.GetCollectorConfig(); collectorConfig != nil {
				return collectorConfig
			}
		}
	}

	return nil
}

func (s *serviceImpl) ProcessMessage(msg *central.MsgToSensor) error {
	log.Info("In ProcessMessage")
	if collectorConfig := getCollectorConfig(msg); collectorConfig != nil {
		log.Infof("Sending message %+v ", collectorConfig)
		s.collectorC <- &sensor.MsgToCollector{
			Msg: &sensor.MsgToCollector_CollectorConfig{
				CollectorConfig: collectorConfig,
			},
		}
	}

	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

type connectionManager struct {
	connectionLock sync.RWMutex
	connectionMap  map[string]sensor.CollectorService_CommunicateServer
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connectionMap: make(map[string]sensor.CollectorService_CommunicateServer),
	}
}

func (c *connectionManager) add(hello *sensor.CollectorHello, connection sensor.CollectorService_CommunicateServer) {
	c.connectionLock.Lock()
	log.Info("Adding connection")
	log.Infof("connection= %+v", connection)
	defer c.connectionLock.Unlock()

	c.connectionMap[hello.GetDeploymentIdentification().GetK8SNodeName()] = true
}

func (c *connectionManager) remove(nodeName string) {
	c.connectionLock.Lock()
	log.Info("Removing collector connection to node: %s", nodeName)
	defer c.connectionLock.Unlock()

	delete(c.connectionMap, nodeName)
}

func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {
	incomingMD := metautils .ExtractIncoming(context.Background())
	incomingMD.Get(CollectorHelloMetadataKey)
	outMD := metautils.MD{}

	collectorSupportsHello := incomingMD.Get(CollectorHelloMetadataKey) == "true" +
	if collectorSupportsHello {
		outMD.Set(CollectorHelloMetadataKey, "true")
	}

	if err := server.SendHeader(metadata.MD(outMD)); err != nil {
		return errors.Wrap(err, "sending header metadata to collector")
	}

	if !collectorSupportsHello {
		return errors.New("Collector's does not support the CollectorHello handshake. It seems Collector is too old, please upgrade your Secured Cluster.")
	}

	firstMsg, err := server.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving first message")
	}

	collectorHello := firstMsg.GetCollectorHello()
	if collectorHello == nil {
		return errors.Wrapf(err, "first message received is not a CollectorHello message, but %T", firstMsg.GetMsg())
	}

	s.connectionManager.add(collectorHello, server)
	defer s.connectionManager.remove(collectorHello.GetDeploymentIdentification().GetK8SNodeName())
	log.Info("In Communicate")

	for msg := range s.collectorC {
		log.Info("Sending message")
		log.Infof("len(s.connectionManager.connectionMap)= %+v", len(s.connectionManager.connectionMap))
		for conn := range s.connectionManager.connectionMap {
			err := conn.Send(msg)
			if err != nil {
				log.Error(err, "Failed sending runtime config to Collector")
				return err
			}
		}
	}

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
