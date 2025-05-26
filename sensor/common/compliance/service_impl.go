package compliance

import (
	"context"
	"fmt"
	"sync/atomic"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/orchestrator"
	"google.golang.org/grpc"
)

// ComplianceService is the struct that manages the compliance results and audit log events
type serviceImpl struct {
	sensor.UnimplementedComplianceServiceServer

	output           chan *compliance.ComplianceReturn
	auditEvents      chan *sensor.AuditEvents
	nodeInventories  chan *storage.NodeInventory
	indexReportWraps chan *index.IndexReportWrap

	complianceC <-chan common.MessageToComplianceWithAddress

	auditLogCollectionManager AuditLogCollectionManager

	orchestrator orchestrator.Orchestrator

	connectionManager *connectionManager

	offlineMode *atomic.Bool
	stopperLock sync.Mutex
	stopper     set.Set[concurrency.Stopper]
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		s.offlineMode.Store(false)
	case common.SensorComponentEventOfflineMode:
		s.offlineMode.Store(true)
	}
}

func (s *serviceImpl) Start() error {
	return nil
}

func (s *serviceImpl) Stop(_ error) {
	concurrency.WithLock(&s.stopperLock, func() {
		for _, stopper := range s.stopper.AsSlice() {
			stopper.Client().Stop()
			_ = stopper.Client().Stopped().Wait()
		}
	})
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (s *serviceImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
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
func (s *serviceImpl) GetScrapeConfig(_ context.Context, nodeName string) (*sensor.MsgToCompliance_ScrapeConfig, error) {
	nodeScrapeConfig, err := s.orchestrator.GetNodeScrapeConfig(nodeName)
	if err != nil {
		return nil, err
	}

	rt, _ := k8sutil.ParseContainerRuntimeString(nodeScrapeConfig.ContainerRuntimeVersion)

	return &sensor.MsgToCompliance_ScrapeConfig{
		ContainerRuntime: rt,
		IsMasterNode:     nodeScrapeConfig.IsMasterNode,
	}, nil
}

func (s *serviceImpl) startSendingLoop(stopper concurrency.Stopper) {
	defer stopper.Flow().ReportStopped()
	for {
		select {
		case <-stopper.Flow().StopRequested():
			return
		case msg, ok := <-s.complianceC:
			if !ok {
				log.Error("the complianceC was closed unexpectedly")
				return
			}
			if err := s.handleSendingMessage(msg); err != nil {
				log.Errorf("Error sending message to compliance: %v", err)
			}
		}
	}
}

func (s *serviceImpl) handleSendingMessage(msg common.MessageToComplianceWithAddress) error {
	if msg.Broadcast {
		s.connectionManager.forEach(func(node string, server sensor.ComplianceService_CommunicateServer) {
			err := server.Send(msg.Msg)
			if err != nil {
				log.Errorf("Error sending broadcast compliance-message to node %q: %v", node, err)
				return
			}
		})
		return nil
	}
	conn, found := s.connectionManager.connectionMap[msg.Hostname]
	if !found {
		return fmt.Errorf("unable to find connection to compliance: %q", msg.Hostname)

	}
	if err := conn.Send(msg.Msg); err != nil {
		return fmt.Errorf("sending message to compliance on node %q: %w", msg.Hostname, err)
	}
	return nil
}

func (s *serviceImpl) RunScrape(msg *sensor.MsgToCompliance) int {
	var count int

	s.connectionManager.forEach(func(node string, server sensor.ComplianceService_CommunicateServer) {
		err := server.Send(msg)
		if err != nil {
			log.Errorf("Error sending compliance request to node %q: %v", node, err)
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
		return errors.New("compliance did not transmit a hostname in initial metadata")
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

	// Set up this node to start collecting audit log events if it's a master node. It may send a message if the feature is already enabled
	if conf.GetIsMasterNode() {
		log.Infof("Adding node %s to list of eligible compliance nodes for audit log collection because it is on a master node", hostname)
		s.auditLogCollectionManager.AddEligibleComplianceNode(hostname, server)
		defer s.auditLogCollectionManager.RemoveEligibleComplianceNode(hostname)
	}

	stopper := concurrency.NewStopper()
	concurrency.WithLock(&s.stopperLock, func() {
		s.stopper.Add(stopper)
	})
	go s.startSendingLoop(stopper)

	for {
		msg, err := server.Recv()
		if err != nil {
			log.Errorf("receiving from compliance %q: %v", hostname, err)
			// Make sure the stopper stops if there is an error with the connection
			stopper.Client().Stop()
			return err
		}
		switch t := msg.Msg.(type) {
		case *sensor.MsgFromCompliance_Return:
			log.Infof("Received compliance return from %q", msg.GetNode())
			s.output <- t.Return
		case *sensor.MsgFromCompliance_AuditEvents:
			// if we are offline we do not send more audit logs to the manager nor the detector.
			// Upon reconnection Central will sync the last state and Sensor will request to Compliance to start
			// sending the audit logs based on that state.
			if s.offlineMode.Load() {
				continue
			}
			s.auditEvents <- t.AuditEvents
			s.auditLogCollectionManager.AuditMessagesChan() <- msg
		case *sensor.MsgFromCompliance_NodeInventory:
			s.nodeInventories <- t.NodeInventory
		case *sensor.MsgFromCompliance_IndexReport:
			log.Infof("Received index report from %q with %d packages",
				msg.GetNode(), len(msg.GetIndexReport().GetContents().GetPackages()))
			s.indexReportWraps <- &index.IndexReportWrap{
				NodeName:    msg.GetNode(),
				IndexReport: t.IndexReport,
			}
		}
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

func (s *serviceImpl) AuditEvents() chan *sensor.AuditEvents {
	return s.auditEvents
}

func (s *serviceImpl) NodeInventories() <-chan *storage.NodeInventory {
	return s.nodeInventories
}

func (s *serviceImpl) IndexReportWraps() <-chan *index.IndexReportWrap {
	return s.indexReportWraps
}
