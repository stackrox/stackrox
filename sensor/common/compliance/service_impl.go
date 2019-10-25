package compliance

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BenchmarkResultsService is the struct that manages the benchmark results API
type serviceImpl struct {
	output       chan *compliance.ComplianceReturn
	orchestrator orchestrators.Orchestrator

	connectionManager *connectionManager
}

type connectionManager struct {
	connectionLock sync.RWMutex
	connectionMap  map[string]sensor.ComplianceService_CommunicateServer
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connectionMap: make(map[string]sensor.ComplianceService_CommunicateServer),
	}
}

func (c *connectionManager) add(node string, connection sensor.ComplianceService_CommunicateServer) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	c.connectionMap[node] = connection
}

func (c *connectionManager) remove(node string) {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	delete(c.connectionMap, node)
}

func (c *connectionManager) forEach(fn func(node string, server sensor.ComplianceService_CommunicateServer)) {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	for node, server := range c.connectionMap {
		fn(node, server)
	}
}

// GetScrapeConfig returns the scrape configuration for the given node name and scrape ID.
func (s *serviceImpl) GetScrapeConfig(ctx context.Context, nodeName string) (*sensor.MsgToCompliance_ScrapeConfig, error) {
	nodeInfo, err := s.orchestrator.GetNode(nodeName)
	if err != nil {
		return nil, err
	}

	rt, _ := k8sutil.ParseContainerRuntimeString(nodeInfo.Status.NodeInfo.ContainerRuntimeVersion)

	return &sensor.MsgToCompliance_ScrapeConfig{
		ContainerRuntime: rt,
	}, nil
}

func (s *serviceImpl) RunScrape(msg *sensor.MsgToCompliance) int {
	var count int

	s.connectionManager.forEach(func(node string, server sensor.ComplianceService_CommunicateServer) {
		err := server.Send(msg)
		if err != nil {
			log.Errorf("error sending compliance request to node %q: %v", node, err)
			return
		}
		count++
	})
	return count
}

func (s *serviceImpl) Communicate(server sensor.ComplianceService_CommunicateServer) error {
	incomingMD := metautils.ExtractIncoming(server.Context())
	hostname := incomingMD.Get("rox-compliance-nodename")
	if hostname == "" {
		return status.Error(codes.Internal, "compliance did not transmit a hostname in initial metadata")
	}

	log.Infof("Received connection from %q", hostname)

	s.connectionManager.add(hostname, server)
	defer s.connectionManager.remove(hostname)

	conf, err := s.GetScrapeConfig(server.Context(), hostname)
	if err != nil {
		log.Errorf("getting scrape config for %q: %v", hostname, err)
		conf = &sensor.MsgToCompliance_ScrapeConfig{
			ContainerRuntime: storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME,
		}
	}

	err = server.Send(&sensor.MsgToCompliance{
		Msg: &sensor.MsgToCompliance_Config{
			Config: conf,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "sending config to %q", hostname)
	}

	for {
		msg, err := server.Recv()
		if err != nil {
			log.Errorf("error receiving from compliance %q: %v", hostname, err)
			return err
		}
		log.Infof("Received compliance return from %q", msg.GetNode())
		s.output <- msg.GetReturn()
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterComplianceServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Output() chan *compliance.ComplianceReturn {
	return s.output
}
