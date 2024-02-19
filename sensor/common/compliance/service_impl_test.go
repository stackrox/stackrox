package compliance

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	mocksCompliance "github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/orchestrator"
	mocksOrchestrator "github.com/stackrox/rox/sensor/common/orchestrator/mocks"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestComplianceService(t *testing.T) {
	suite.Run(t, new(complianceServiceSuite))
}

type complianceServiceSuite struct {
	suite.Suite
	mockCtrl            *gomock.Controller
	mockOrchestrator    *mocksOrchestrator.MockOrchestrator
	mockAuditLogManager *mocksCompliance.MockAuditLogCollectionManager
	srv                 *serviceImpl
	complianceC         chan common.MessageToComplianceWithAddress
	auditEventInput     chan *sensor.AuditEvents
	mockAuditLogC       chan *sensor.MsgFromCompliance
	stream              sensor.ComplianceService_CommunicateClient
	stopServerFn        func()
}

var _ suite.SetupTestSuite = (*complianceServiceSuite)(nil)
var _ suite.TearDownTestSuite = (*complianceServiceSuite)(nil)

func (s *complianceServiceSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockOrchestrator = mocksOrchestrator.NewMockOrchestrator(s.mockCtrl)
	s.mockAuditLogManager = mocksCompliance.NewMockAuditLogCollectionManager(s.mockCtrl)
	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)
	s.complianceC = make(chan common.MessageToComplianceWithAddress)
	s.auditEventInput = make(chan *sensor.AuditEvents)
	s.mockAuditLogC = make(chan *sensor.MsgFromCompliance)
	s.srv = &serviceImpl{
		output:                    make(chan *compliance.ComplianceReturn),
		nodeInventories:           make(chan *storage.NodeInventory),
		complianceC:               s.complianceC,
		orchestrator:              s.mockOrchestrator,
		auditEvents:               s.auditEventInput,
		auditLogCollectionManager: s.mockAuditLogManager,
		connectionManager:         newConnectionManager(),
		offlineMode:               offlineMode,
	}
	s.createMockService()
}

func (s *complianceServiceSuite) TearDownTest() {
	if s.stopServerFn != nil {
		s.stopServerFn()
	}
}

func (s *complianceServiceSuite) TestServiceOfflineMode() {
	events := []func(){
		s.online,
		s.sendAuditEvent,
		s.readAuditEvent,
		s.offline,
		s.sendAuditEvent,
		s.notReadAuditEvent,
		s.online,
		s.notReadAuditEvent,
		s.sendAuditEvent,
		s.readAuditEvent,
	}
	for _, event := range events {
		event()
	}
}

func (s *complianceServiceSuite) online() {
	s.srv.Notify(common.SensorComponentEventCentralReachable)
}

func (s *complianceServiceSuite) offline() {
	s.srv.Notify(common.SensorComponentEventOfflineMode)
}

func (s *complianceServiceSuite) sendAuditEvent() {
	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_AuditEvents{
			AuditEvents: &sensor.AuditEvents{
				Events: []*storage.KubernetesEvent{
					{
						Id: "1",
					},
				},
			},
		},
	}))
}

func (s *complianceServiceSuite) readAuditEvent() {
	select {
	case event, ok := <-s.auditEventInput:
		s.Require().True(ok)
		s.Require().NotNil(event)
	case <-time.After(500 * time.Millisecond):
		s.Fail("a message to the detector should've been sent")
	}
	select {
	case event, ok := <-s.mockAuditLogC:
		s.Require().True(ok)
		s.Require().NotNil(event)
	case <-time.After(500 * time.Millisecond):
		s.Fail("a message to the audit log manager should've been sent")
	}
}

func (s *complianceServiceSuite) notReadAuditEvent() {
	select {
	case <-s.auditEventInput:
		s.Fail("an unexpected message to the detector was sent")
	case <-time.After(500 * time.Millisecond):
		break
	}
	select {
	case <-s.mockAuditLogC:
		s.Fail("an unexpected message to the audit log manager was sent")
	case <-time.After(500 * time.Millisecond):
		break
	}
}

func (s *complianceServiceSuite) createMockService() {
	s.mockOrchestrator.EXPECT().GetNodeScrapeConfig(gomock.Any()).Times(1).DoAndReturn(func(_ any) (*orchestrator.NodeScrapeConfig, error) {
		return &orchestrator.NodeScrapeConfig{
			ContainerRuntimeVersion: "containerd://1.4.2",
			IsMasterNode:            true,
		}, nil
	})
	s.mockAuditLogManager.EXPECT().AddEligibleComplianceNode(gomock.Any(), gomock.Any()).AnyTimes()
	s.mockAuditLogManager.EXPECT().RemoveEligibleComplianceNode(gomock.Any()).AnyTimes()
	s.mockAuditLogManager.EXPECT().AuditMessagesChan().AnyTimes().DoAndReturn(func() chan<- *sensor.MsgFromCompliance {
		return s.mockAuditLogC
	})

	// Create a grpc server
	ctx, cancel := context.WithCancel(context.Background())
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	grpcServer := grpc.NewServer()
	sensor.RegisterComplianceServiceServer(grpcServer, s.srv)
	go func() {
		utils.IgnoreError(func() error {
			return grpcServer.Serve(listener)
		})
	}()

	// Create the client stream
	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(s.T(), err)
	cli := sensor.NewComplianceServiceClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, "rox-compliance-nodename", "fake-compliance")
	stream, err := cli.Communicate(ctx)
	require.NoError(s.T(), err)
	s.stream = stream

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// To not timeout if the server send a message we read here.
				_, _ = s.stream.Recv()
			}
		}
	}()

	s.stopServerFn = func() {
		cancel()
		utils.IgnoreError(listener.Close)
		grpcServer.Stop()
	}
}
