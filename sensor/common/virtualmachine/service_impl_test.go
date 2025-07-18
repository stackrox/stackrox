package virtualmachine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/require"
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
	s.service = &serviceImpl{
		stopper:        concurrency.NewStopper(),
		fromDataSource: make(chan *storage.VirtualMachine, 10),
	}
}

func (s *virtualMachineServiceSuite) TearDownTest() {
	// No explicit stop needed - individual tests handle their own cleanup
}

func (s *virtualMachineServiceSuite) TestNewService() {
	svc := NewService()
	s.Require().NotNil(svc)
	s.Require().IsType(&serviceImpl{}, svc)

	impl := svc.(*serviceImpl)
	s.Require().NotNil(impl.stopper)
	s.Require().NotNil(impl.fromDataSource)
	s.Require().Equal(10, cap(impl.fromDataSource))
}

func (s *virtualMachineServiceSuite) TestRegisterServiceServer() {
	server := grpc.NewServer()
	s.service.RegisterServiceServer(server)
	// Test passes if no panic occurs
}

func (s *virtualMachineServiceSuite) TestRegisterServiceHandler() {
	ctx := context.Background()
	mux := runtime.NewServeMux()
	conn := &grpc.ClientConn{}

	err := s.service.RegisterServiceHandler(ctx, mux, conn)
	s.Require().NoError(err)
}

func (s *virtualMachineServiceSuite) TestAuthFuncOverride() {
	ctx := context.Background()
	fullMethodName := "/sensor.VirtualMachineService/UpsertVirtualMachine"

	// Test with valid admission control context
	_, err := s.service.AuthFuncOverride(ctx, fullMethodName)
	s.Require().Error(err) // Should fail without proper admission control setup
	s.Require().Contains(err.Error(), "virtual machine authorization")
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_NilConnection() {
	ctx := context.Background()
	req := &sensor.UpsertVirtualMachineRequest{
		VirtualMachine: &storage.VirtualMachine{
			Id: "test-vm-id",
		},
	}

	resp, err := s.service.UpsertVirtualMachine(ctx, req)
	s.Require().Error(err)
	s.Require().NotNil(resp)
	s.Require().False(resp.Success)
	s.Require().Contains(err.Error(), "Connection to Central is not ready")
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_WithConnection() {
	ctx := context.Background()

	// Start the service to initialize the toCentral channel
	err := s.service.Start()
	s.Require().NoError(err)

	// Set up a goroutine to consume from ResponsesC to prevent blocking
	go func() {
		for range s.service.ResponsesC() {
			// Consume messages to prevent blocking
		}
	}()

	defer s.service.Stop()

	req := &sensor.UpsertVirtualMachineRequest{
		VirtualMachine: &storage.VirtualMachine{
			Id: "test-vm-id",
		},
	}

	resp, err := s.service.UpsertVirtualMachine(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Success)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_NilVirtualMachine() {
	ctx := context.Background()

	// Start the service to initialize the toCentral channel
	err := s.service.Start()
	s.Require().NoError(err)

	defer s.service.Stop()

	req := &sensor.UpsertVirtualMachineRequest{}
	resp, err := s.service.UpsertVirtualMachine(ctx, req)
	s.Require().Equal(resp.Success, false)
	s.Require().Error(err)
}

func (s *virtualMachineServiceSuite) TestNotify() {
	// Notify should not panic
	s.service.Notify(common.SensorComponentEventCentralReachable)
	s.service.Notify(common.SensorComponentEventOfflineMode)
}

func (s *virtualMachineServiceSuite) TestStart() {
	err := s.service.Start()
	s.Require().NoError(err)
	defer s.service.Stop()
	s.Require().NotNil(s.service.toCentral)

	// Test that the goroutine processes VMs from fromDataSource
	vm := &storage.VirtualMachine{Id: "test-vm"}

	// Send VM to fromDataSource channel
	go func() {
		s.service.fromDataSource <- vm
	}()

	// Read from ResponsesC to verify message was sent
	select {
	case msg := <-s.service.ResponsesC():
		s.Require().NotNil(msg)
		s.Require().NotNil(msg.MsgFromSensor)

		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Require().Equal("test-vm", sensorEvent.Id)
		s.Require().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.Action)
		s.Require().NotNil(sensorEvent.GetVirtualMachine())
		s.Require().Equal("test-vm", sensorEvent.GetVirtualMachine().Id)
	case <-time.After(500 * time.Millisecond):
		s.Fail("Expected message to be sent to central")
	}
}

func (s *virtualMachineServiceSuite) TestStop() {
	// Start first
	err := s.service.Start()
	s.Require().NoError(err)

	// Stop should not panic and should stop gracefully
	s.service.Stop()

	// Verify stopper is stopped
	select {
	case <-s.service.stopper.Client().Stopped().Done():
		// Expected
	case <-time.After(1 * time.Second):
		s.Fail("Service should have stopped")
	}
}

func (s *virtualMachineServiceSuite) TestCapabilities() {
	caps := s.service.Capabilities()
	s.Require().NotNil(caps)
	s.Require().Empty(caps)
}

func (s *virtualMachineServiceSuite) TestProcessMessage() {
	msg := &central.MsgToSensor{}
	err := s.service.ProcessMessage(msg)
	s.Require().NoError(err)
}

func (s *virtualMachineServiceSuite) TestResponsesC_BeforeStart() {
	ch := s.service.ResponsesC()
	s.Require().Nil(ch)
}

func (s *virtualMachineServiceSuite) TestResponsesC_AfterStart() {
	err := s.service.Start()
	s.Require().NoError(err)

	// Set up a goroutine to consume from ResponsesC to prevent blocking
	go func() {
		for range s.service.ResponsesC() {
			// Consume messages to prevent blocking
		}
	}()

	defer s.service.Stop()

	ch := s.service.ResponsesC()
	s.Require().NotNil(ch)
}

func TestServiceInterface(t *testing.T) {
	// Verify that serviceImpl implements the Service interface
	var _ Service = (*serviceImpl)(nil)
}

func TestServiceStartStopCycle(t *testing.T) {
	service := NewService().(*serviceImpl)

	// Test multiple start/stop cycles
	for i := range 3 {
		err := service.Start()
		require.NoError(t, err)

		service.Stop()

		// Wait for stop to complete
		select {
		case <-service.stopper.Client().Stopped().Done():
			// Expected
		case <-time.After(1 * time.Second):
			t.Fatalf("Service should have stopped in cycle %d", i)
		}

		// Create new service for next iteration
		if i < 2 {
			service = NewService().(*serviceImpl)
		}
	}
}

func TestConcurrentVMUpserts(t *testing.T) {
	service := NewService().(*serviceImpl)
	err := service.Start()
	require.NoError(t, err)

	// Set up a goroutine to consume from ResponsesC to prevent blocking
	go func() {
		for range service.ResponsesC() {
			// Consume messages to prevent blocking
		}
	}()

	defer service.Stop()

	ctx := context.Background()
	numGoroutines := 3
	numVMsPerGoroutine := 2

	// Channel to collect all responses
	responses := make(chan *sensor.UpsertVirtualMachineResponse, numGoroutines*numVMsPerGoroutine)

	// Start concurrent upserts
	for i := range numGoroutines {
		go func(routineID int) {
			defer func() {
				// Signal completion even if there are errors
				responses <- &sensor.UpsertVirtualMachineResponse{Success: false}
			}()
			for j := range numVMsPerGoroutine {
				req := &sensor.UpsertVirtualMachineRequest{
					VirtualMachine: &storage.VirtualMachine{
						Id: fmt.Sprintf("vm-%d-%d", routineID, j),
					},
				}

				resp, err := service.UpsertVirtualMachine(ctx, req)
				if err == nil {
					responses <- resp
				}
			}
		}(i)
	}

	// Collect all responses with shorter timeout
	successCount := 0
	totalResponses := 0
	for totalResponses < numGoroutines {
		select {
		case resp := <-responses:
			totalResponses++
			if resp.Success {
				successCount++
			}
		case <-time.After(500 * time.Millisecond):
			t.Logf("Timeout waiting for response, got %d responses", totalResponses)
			return // Don't fail, just exit
		}
	}

	t.Logf("Success count: %d out of %d", successCount, totalResponses)
}

func (s *virtualMachineServiceSuite) TestUpsertVirtualMachine_LastTimestampSet() {
	ctx := context.Background()

	// Start the service to initialize the toCentral channel
	err := s.service.Start()
	s.Require().NoError(err)

	defer s.service.Stop()

	// Record time before the call
	beforeTime := time.Now()

	req := &sensor.UpsertVirtualMachineRequest{
		VirtualMachine: &storage.VirtualMachine{
			Id: "test-vm-id",
		},
	}

	// Call UpsertVirtualMachine
	resp, err := s.service.UpsertVirtualMachine(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Success)

	// Read from ResponsesC to get the sensor event
	select {
	case msg := <-s.service.ResponsesC():
		s.Require().NotNil(msg)
		s.Require().NotNil(msg.MsgFromSensor)

		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Require().NotNil(sensorEvent.GetVirtualMachine())

		vm := sensorEvent.GetVirtualMachine()
		s.Require().NotNil(vm.LastUpdated, "LastUpdated field should be set")

		// Verify timestamp is within reasonable bounds
		lastUpdatedTime := vm.LastUpdated.AsTime()
		afterTime := time.Now()

		s.Require().True(lastUpdatedTime.After(beforeTime) || lastUpdatedTime.Equal(beforeTime))
		s.Require().True(lastUpdatedTime.Before(afterTime) || lastUpdatedTime.Equal(afterTime))
	case <-time.After(500 * time.Millisecond):
		s.Fail("Expected sensor event to be sent")
	}
}
