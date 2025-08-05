package virtualmachine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
)

func TestVirtualMachineComponent(t *testing.T) {
	suite.Run(t, new(virtualMachineComponentSuite))
}

type virtualMachineComponentSuite struct {
	suite.Suite
	component *componentImpl
}

func (s *virtualMachineComponentSuite) SetupTest() {
	s.component = &componentImpl{
		centralReady:    concurrency.NewSignal(),
		lock:            &sync.Mutex{},
		stopper:         concurrency.NewStopper(),
		virtualMachines: make(chan *storage.VirtualMachine),
	}
}

func (s *virtualMachineComponentSuite) TestSend() {
	err := s.component.Start()
	s.Require().NoError(err)
	s.component.Notify(common.SensorComponentEventCentralReachable)
	defer s.component.Stop()
	s.Require().NotNil(s.component.toCentral)

	// Test that the goroutine processes sent VMs.
	vm := &storage.VirtualMachine{Id: "test-vm"}
	go s.component.Send(context.Background(), vm)

	// Read from ResponsesC to verify message was sent.
	select {
	case msg := <-s.component.ResponsesC():
		s.Require().NotNil(msg)
		s.Require().NotNil(msg.MsgFromSensor)

		sensorEvent := msg.GetEvent()
		s.Require().NotNil(sensorEvent)
		s.Assert().Equal("test-vm", sensorEvent.GetId())
		s.Assert().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.Action)
		s.Assert().NotNil(sensorEvent.GetVirtualMachine())
		s.Assert().Equal("test-vm", sensorEvent.GetVirtualMachine().GetId())
	case <-time.After(time.Second):
		s.Fail("Expected message to be sent to central")
	}
}

func (s *virtualMachineComponentSuite) TestStop() {
	err := s.component.Start()
	s.Require().NoError(err)

	// Stop should not panic and should stop gracefully.
	s.component.Stop()

	// Verify stopper is stopped.
	select {
	case <-s.component.stopper.Client().Stopped().Done():
		// Expected.
	case <-time.After(time.Second):
		s.Fail("Component should have stopped")
	}
}

func (s *virtualMachineComponentSuite) TestCapabilities() {
	caps := s.component.Capabilities()
	s.Require().Empty(caps)
}

func (s *virtualMachineComponentSuite) TestProcessMessage() {
	msg := &central.MsgToSensor{}
	err := s.component.ProcessMessage(context.Background(), msg)
	s.Require().NoError(err)
}

func (s *virtualMachineComponentSuite) TestResponsesC_BeforeStart() {
	s.Assert().Panics(func() { _ = s.component.ResponsesC() })
}

func (s *virtualMachineComponentSuite) TestResponsesC_AfterStart() {
	err := s.component.Start()
	s.Require().NoError(err)
	defer s.component.Stop()

	ch := s.component.ResponsesC()
	s.Require().NotNil(ch)
}

func (s *virtualMachineComponentSuite) TestConcurrentSends() {
	err := s.component.Start()
	s.Require().NoError(err)
	s.component.Notify(common.SensorComponentEventCentralReachable)
	defer s.component.Stop()

	ctx := context.Background()
	numGoroutines := 3
	numVMsPerGoroutine := 2

	// Start concurrent sends.
	for i := range numGoroutines {
		go func(routineID int) {
			for j := range numVMsPerGoroutine {
				req := &storage.VirtualMachine{
					Id: fmt.Sprintf("vm-%d-%d", routineID, j),
				}
				err := s.component.Send(ctx, req)
				s.Require().NoError(err)
			}
		}(i)
	}

	// Collect all responses with shorter timeout.
	totalResponses := 0
	for range numGoroutines * numVMsPerGoroutine {
		select {
		case <-s.component.toCentral:
			totalResponses++
		case <-time.After(500 * time.Millisecond):
			s.T().Logf("Timeout waiting for response, got %d responses", totalResponses)
			return // Don't fail, just exit
		}
	}
	s.Assert().Equal(numGoroutines*numVMsPerGoroutine, totalResponses)
}
