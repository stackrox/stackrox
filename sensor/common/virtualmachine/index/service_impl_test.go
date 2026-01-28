package index

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	"github.com/stretchr/testify/require"
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
	s.service = &serviceImpl{
		handler: NewHandler(nil, s.store),
		store:   s.store,
	}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineServiceSuite) TestNewService() {
	svc := NewService(NewHandler(nil, s.store), s.store)
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
			VsockCid: "42",
		},
	}

	resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
	s.Assert().NotNil(resp)
	s.Assert().False(resp.GetSuccess())
	s.Assert().Error(err)
	s.Assert().ErrorIs(err, errox.InvariantViolation)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_WithConnection() {
	ctx := context.Background()

	// Start the handler to initialize the toCentral channel.
	err := s.service.handler.Start()
	s.Require().NoError(err)
	defer s.service.handler.Stop()
	s.service.handler.Notify(common.SensorComponentEventCentralReachable)
	s.store.EXPECT().GetFromCID(uint32(42)).Times(1).Return(&virtualmachine.Info{
		ID: "test-vm",
	})

	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "42",
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

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_InvalidVsockCID() {
	ctx := context.Background()
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "invalid-vsock-cid",
		},
	}

	resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
	s.Require().NotNil(resp)
	s.Require().False(resp.GetSuccess())
	s.Require().ErrorIs(err, errox.InvalidArgs)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_ShouldNotPanicWhenDiscoveredDataMissing() {
	ctx := context.Background()
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: &v1.IndexReport{
			VsockCid: "42",
		},
	}

	s.NotPanics(func() {
		resp, err := s.service.UpsertVirtualMachineIndexReport(ctx, req)
		s.Require().NotNil(resp)
		s.Require().Error(err)
	})
}

func TestUpsertVirtualMachineIndexReport_DiscoveredFactsUpdate(t *testing.T) {
	t.Setenv(features.VirtualMachines.EnvVar(), "true")
	t.Run("should pass discovered data to handler", func(it *testing.T) {
		ctrl := gomock.NewController(it)
		store := mocks.NewMockVirtualMachineStore(ctrl)
		handler := mocks.NewMockHandler(ctrl)
		service := &serviceImpl{
			handler: handler,
			store:   store,
		}

		// Service now just passes discovered data to handler.Send
		// Handler handles all store operations and update emission
		handler.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, report *v1.IndexReport, data *v1.DiscoveredData) {
				require.NotNil(it, report)
				require.NotNil(it, data)
				require.Equal(it, "42", report.GetVsockCid())
			}).
			Return(nil)

		req := &sensor.UpsertVirtualMachineIndexReportRequest{
			IndexReport: &v1.IndexReport{
				VsockCid: "42",
			},
			DiscoveredData: &v1.DiscoveredData{
				DetectedOs:        v1.DetectedOS_RHEL,
				OsVersion:         "9.4",
				ActivationStatus:  v1.ActivationStatus_ACTIVE,
				DnfMetadataStatus: v1.DnfMetadataStatus_AVAILABLE,
			},
		}

		resp, err := service.UpsertVirtualMachineIndexReport(context.Background(), req)
		require.NoError(it, err)
		require.True(it, resp.GetSuccess())
	})
}
