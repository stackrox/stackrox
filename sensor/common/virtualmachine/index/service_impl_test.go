package index

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func TestVirtualMachineService(t *testing.T) {
	suite.Run(t, new(virtualMachineServiceSuite))
}

type virtualMachineServiceSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	store   *mocks.MockVirtualMachineStore
	service *serviceImpl
}

func (s *virtualMachineServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.store = mocks.NewMockVirtualMachineStore(s.ctrl)
	s.service = &serviceImpl{handler: NewHandler(s.store)}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineServiceSuite) TestNewService() {
	svc := NewService(NewHandler(s.store), nil)
	s.Require().NotNil(svc)
	s.Require().IsType(&serviceImpl{}, svc)

	impl, ok := svc.(*serviceImpl)
	s.Assert().NotNil(impl)
	s.Assert().True(ok)
}

func (s *virtualMachineServiceSuite) TestRegisterServiceServer() {
	server := grpc.NewServer()
	s.service.RegisterServiceServer(server)
	// Test passes if no panic occurs.
}

func (s *virtualMachineServiceSuite) TestRegisterServiceHandler() {
	ctx := context.Background()
	mux := runtime.NewServeMux()
	conn := &grpc.ClientConn{}

	err := s.service.RegisterServiceHandler(ctx, mux, conn)
	s.Assert().NoError(err)
}

func (s *virtualMachineServiceSuite) TestAuthFuncOverride() {
	ctx := context.Background()
	fullMethodName := "/sensor.VirtualMachineIndexReportService/UpsertVirtualMachineIndexReport"

	_, err := s.service.AuthFuncOverride(ctx, fullMethodName)
	s.Assert().Error(err) // Should fail without proper collector setup
	s.Assert().Contains(err.Error(), "virtual machine index report authorization")
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_NilConnection() {
	ctx := context.Background()
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "test-vm-id",
		},
	}

	resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
	s.Assert().NotNil(resp)
	s.Assert().False(resp.GetSuccess())
	s.Assert().Error(err)
	s.Assert().ErrorIs(err, errox.ResourceExhausted)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_WithConnection() {
	ctx := context.Background()

	// Start the handler to initialize the toCentral channel.
	err := s.service.handler.Start()
	s.Require().NoError(err)
	defer s.service.handler.Stop()
	s.service.handler.Notify(common.SensorComponentEventCentralReachable)

	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "test-vm-id",
		},
	}

	resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
	s.Require().NotNil(resp)
	s.Require().True(resp.GetSuccess())
	s.Require().NoError(err)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_NilVirtualMachine() {
	ctx := context.Background()

	// Start the handler to initialize the toCentral channel.
	err := s.service.handler.Start()
	s.Require().NoError(err)
	defer s.service.handler.Stop()
	s.service.handler.Notify(common.SensorComponentEventCentralReachable)

	req := &sensor.UpsertVirtualMachineIndexReportRequest{}
	resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
	s.Require().Equal(resp.GetSuccess(), false)
	s.Require().ErrorIs(err, errox.InvalidArgs)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_ShouldNotPanicWhenDiscoveredDataMissing() {
	ctx := context.Background()
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "test-vm-id",
		},
	}

	s.NotPanics(func() {
		resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
		s.Require().NotNil(resp)
		s.Require().Error(err)
	})
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_PushSuppressedForActivelyScrapedVM() {
	ctx := context.Background()

	var sendCalled bool
	mockHandler := mocks.NewMockHandler(s.ctrl)
	mockHandler.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *v1.IndexReport) error {
			sendCalled = true
			return nil
		},
	)

	svc := &serviceImpl{
		handler: mockHandler,
		// Mark VM with CID 100 as actively scraped via pull mode.
		pullChecker: &fakePullChecker{scraped: map[string]bool{"100": true}},
	}

	// CID 100 has both push and pull active simultaneously. When both modes
	// coexist, pull takes precedence and the push report must be suppressed
	// to avoid sending duplicate data to Central.
	// (The pull path forwarding is tested separately in scraper_test.go.)
	resp, err := svc.UpsertVirtualMachineIndexReport(ctx, &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{VsockCid: "100"},
	})
	s.Require().NoError(err)
	s.Assert().True(resp.GetSuccess())
	s.Assert().False(sendCalled, "Send must NOT be called when pull is active for this VM")

	// CID 200 uses push only (not being pulled). Its push report must be
	// forwarded to Central as usual.
	resp, err = svc.UpsertVirtualMachineIndexReport(ctx, &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{VsockCid: "200"},
	})
	s.Require().NoError(err)
	s.Assert().True(resp.GetSuccess())
	s.Assert().True(sendCalled, "Send MUST be called when push is the only mode for this VM")
}

type fakePullChecker struct {
	scraped map[string]bool
}

func (f *fakePullChecker) IsActivelyScraped(key string) bool {
	return f.scraped[key]
}
