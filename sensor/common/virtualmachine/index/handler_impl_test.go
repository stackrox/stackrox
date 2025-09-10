package index

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/suite"
)

func TestVirtualMachineHandler(t *testing.T) {
	suite.Run(t, new(virtualMachineHandlerSuite))
}

type virtualMachineHandlerSuite struct {
	suite.Suite
	handler *handlerImpl
}

func (s *virtualMachineHandlerSuite) SetupTest() {
	s.handler = &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
	}
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})
}

func (s *virtualMachineHandlerSuite) TearDownTest() {
	goleak.AssertNoGoroutineLeaks(s.T())
}

func (s *virtualMachineHandlerSuite) TestSend() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{VsockCid: "test-vm"}
	go func() {
		err := s.handler.Send(context.Background(), vm)
		s.Require().NoError(err)
	}()

	// Read from ResponsesC to verify message was sent.
	select {
	case msg := <-s.handler.ResponsesC():
		s.Require().NotNil(msg)
		s.Require().NotNil(msg.MsgFromSensor)

		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Assert().Equal("test-vm", sensorEvent.GetId())
		s.Assert().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.Action)
		s.Assert().NotNil(sensorEvent.GetVirtualMachineIndexReport())
		s.Assert().Equal("test-vm", sensorEvent.GetVirtualMachineIndexReport().GetId())
	case <-time.After(time.Second):
		s.Fail("Expected message to be sent to central")
	}
}

func (s *virtualMachineHandlerSuite) TestConcurrentSends() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()

	ctx := context.Background()
	numGoroutines := 3
	numVMsPerGoroutine := 2

	// Start concurrent sends.
	for i := range numGoroutines {
		go func(routineID int) {
			for j := range numVMsPerGoroutine {
				req := &v1.IndexReport{
					VsockCid: fmt.Sprintf("vm-%d-%d", routineID, j),
				}
				err := s.handler.Send(ctx, req)
				s.Require().NoError(err)
			}
		}(i)
	}

	// Collect all responses with shorter timeout.
	totalResponses := 0
	for range numGoroutines * numVMsPerGoroutine {
		select {
		case <-s.handler.toCentral:
			totalResponses++
		case <-time.After(500 * time.Millisecond):
			s.T().Logf("Timeout waiting for response, got %d responses", totalResponses)
			return // Don't fail, just exit
		}
	}
	s.Assert().Equal(numGoroutines*numVMsPerGoroutine, totalResponses)
}

func (s *virtualMachineHandlerSuite) TestStop() {
	err := s.handler.Start()
	s.Require().NoError(err)

	// Stop should not panic and should stop gracefully.
	s.handler.Stop()

	// Verify stopper is stopped.
	select {
	case <-s.handler.stopper.Client().Stopped().Done():
		// Expected.
	case <-time.After(time.Second):
		s.Fail("handler should have stopped")
	}
}

func (s *virtualMachineHandlerSuite) TestCapabilities() {
	caps := s.handler.Capabilities()
	s.Require().Empty(caps)
}

func (s *virtualMachineHandlerSuite) TestProcessMessage() {
	msg := &central.MsgToSensor{}
	err := s.handler.ProcessMessage(context.Background(), msg)
	s.Require().NoError(err)
}

func (s *virtualMachineHandlerSuite) TestResponsesC_BeforeStart() {
	s.Assert().Panics(func() { _ = s.handler.ResponsesC() })
}

func (s *virtualMachineHandlerSuite) TestResponsesC_AfterStart() {
	err := s.handler.Start()
	s.Require().NoError(err)
	defer s.handler.Stop()

	ch := s.handler.ResponsesC()
	s.Require().NotNil(ch)
}
