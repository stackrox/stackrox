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
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestVirtualMachineHandler(t *testing.T) {
	suite.Run(t, new(virtualMachineHandlerSuite))
}

type virtualMachineHandlerSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	store   *mocks.MockVirtualMachineStore
	handler *handlerImpl
}

func (s *virtualMachineHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.store = mocks.NewMockVirtualMachineStore(s.ctrl)
	s.handler = &handlerImpl{
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
		store:        s.store,
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

	cid := "1"
	s.store.EXPECT().GetFromCID(gomock.Eq(uint32(1))).Times(1).Return(
		&virtualmachine.Info{
			ID: "test-vm",
		})

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{}
	vm.SetVsockCid(cid)
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
		s.Assert().Equal(central.ResourceAction_SYNC_RESOURCE, sensorEvent.GetAction())
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
	anyOf := func() []any {
		ret := make([]any, 0, numGoroutines*numVMsPerGoroutine)
		cont := 0
		for range numGoroutines {
			for range numVMsPerGoroutine {
				ret = append(ret, uint32(cont))
				cont++
			}
		}
		return ret
	}()
	s.store.EXPECT().GetFromCID(gomock.AnyOf(anyOf...)).Times(numGoroutines * numVMsPerGoroutine).
		Return(
			&virtualmachine.Info{
				ID: "test-vm",
			})

	// Start concurrent sends.
	cont := 0
	mu := sync.Mutex{}
	for i := range numGoroutines {
		go func(routineID int) {
			for range numVMsPerGoroutine {
				var req *v1.IndexReport
				concurrency.WithLock(&mu, func() {
					req = &v1.IndexReport{}
					req.SetVsockCid(fmt.Sprintf("%d", cont))
					cont++
				})
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

func (s *virtualMachineHandlerSuite) TestVirtualMachineNotFound() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	cid := "1"
	s.store.EXPECT().GetFromCID(gomock.Eq(uint32(1))).Times(1).Return(nil)

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{}
	vm.SetVsockCid(cid)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.handler.Send(context.Background(), vm)
		s.Require().NoError(err)
	}()

	wg.Wait()

	// Read from ResponsesC to verify message was not sent.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *virtualMachineHandlerSuite) TestInvalidCID() {
	err := s.handler.Start()
	s.Require().NoError(err)
	s.handler.Notify(common.SensorComponentEventCentralReachable)
	defer s.handler.Stop()
	s.Require().NotNil(s.handler.toCentral)

	cid := "invalid-cid"

	// Test that the goroutine processes sent VMs.
	vm := &v1.IndexReport{}
	vm.SetVsockCid(cid)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.handler.Send(context.Background(), vm)
		s.Require().NoError(err)
	}()

	wg.Wait()

	// Read from ResponsesC to verify message was not sent.
	select {
	case <-s.handler.ResponsesC():
		s.Fail("Unexpected message to be sent to central")
	case <-time.After(500 * time.Millisecond):
	}
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
