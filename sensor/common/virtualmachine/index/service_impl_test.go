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
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

func TestVirtualMachineService(t *testing.T) {
	suite.Run(t, new(virtualMachineServiceSuite))
}

type virtualMachineServiceSuite struct {
	suite.Suite
	service *serviceImpl
}

func (s *virtualMachineServiceSuite) SetupTest() {
	s.service = &serviceImpl{handler: NewHandler()}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineServiceSuite) TestNewService() {
	svc := NewService(NewHandler())
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
	s.Assert().False(resp.Success)
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
	s.Require().True(resp.Success)
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
	s.Require().Equal(resp.Success, false)
	s.Require().ErrorIs(err, errox.InvalidArgs)
}
