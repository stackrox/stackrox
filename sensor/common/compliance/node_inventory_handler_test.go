package compliance

import (
	"errors"
	"fmt"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

func TestNodeInventoryHandler(t *testing.T) {
	suite.Run(t, &NodeInventoryHandlerTestSuite{})
}

func fakeNodeInventory(nodeName string) *storage.NodeInventory {
	msg := &storage.NodeInventory{
		NodeId:   uuid.Nil.String(),
		NodeName: nodeName,
		ScanTime: timestamp.TimestampNow(),
		Components: &storage.NodeInventory_Components{
			Namespace: "rhcos:4.11",
			RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:        int64(1),
					Name:      "vim-minimal",
					Namespace: "rhel:8",
					Version:   "2:7.4.629-6.el8",
					Arch:      "x86_64",
					Module:    "",
					AddedBy:   "hardcoded",
				},
			},
			RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
		},
		Notes: []storage.NodeInventory_Note{storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE},
	}
	return msg
}

var _ suite.TearDownTestSuite = (*NodeInventoryHandlerTestSuite)(nil)

type NodeInventoryHandlerTestSuite struct {
	suite.Suite
}

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
	)
}

func (s *NodeInventoryHandlerTestSuite) TearDownTest() {
	assertNoGoroutineLeaks(s.T())
}

func (s *NodeInventoryHandlerTestSuite) TestCapabilities() {
	inventories := make(chan *storage.NodeInventory)
	defer close(inventories)
	h := NewNodeInventoryHandler(inventories, &mockAlwaysHitNodeIDMatcher{})
	s.Nil(h.Capabilities())
}

func (s *NodeInventoryHandlerTestSuite) TestResponsesCShouldPanicWhenNotStarted() {
	inventories := make(chan *storage.NodeInventory)
	defer close(inventories)
	h := NewNodeInventoryHandler(inventories, &mockAlwaysHitNodeIDMatcher{})
	s.Panics(func() {
		h.ResponsesC()
	})
}

// TestStopHandler goal is to stop handler while there are still some messages to process
// in the channel passed into NewNodeInventoryHandler.
// We expect that premature stop of the handler results in a clean stop without any race conditions or goroutine leaks.
// Exec with: go test -race -count=1 -v -run ^TestNodeInventoryHandler$ ./sensor/common/compliance
func (s *NodeInventoryHandlerTestSuite) TestStopHandler() {
	inventories := make(chan *storage.NodeInventory)
	defer close(inventories)
	producer := concurrency.NewStopper()
	h := NewNodeInventoryHandler(inventories, &mockAlwaysHitNodeIDMatcher{})
	s.NoError(h.Start())
	h.Notify(common.SensorComponentEventCentralReachable)
	consumer := consumeAndCount(h.ResponsesC(), 1)
	// This is a producer that stops the handler after producing the first message and then sends many (29) more messages.
	go func() {
		defer producer.Flow().ReportStopped()
		for i := 0; i < 30; i++ {
			select {
			case <-producer.Flow().StopRequested():
				return
			case inventories <- fakeNodeInventory("Node"):
				if i == 0 {
					s.NoError(consumer.Stopped().Wait()) // This blocks until consumer receives its 1 message
					h.Stop(nil)
				}
			}
		}
	}()

	s.NoError(h.Stopped().Wait())

	producer.Client().Stop()
	s.NoError(producer.Client().Stopped().Wait())
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerRegularRoutine() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerStopIgnoresError() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	errTest := errors.New("example-stop-error")
	h.Stop(errTest)
	// This test indicates that the handler ignores an error that's supplied to its Stop function.
	// The handler will report either an internal error if it occurred during processing or nil otherwise.
	s.NoError(h.Stopped().Wait())
}

type testState struct {
	event             common.SensorComponentEvent
	expectedACKCount  int
	expectedNACKCount int
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerCentralACKsToCompliance() {
	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	s.NoError(h.Start())
	h.Notify(common.SensorComponentEventCentralReachable)

	cases := map[string]struct {
		centralReply      central.NodeInventoryACK_Action
		expectedACKCount  int
		expectedNACKCount int
	}{
		"Central ACK should be forwarded to Compliance": {
			centralReply:      central.NodeInventoryACK_ACK,
			expectedACKCount:  1,
			expectedNACKCount: 0,
		},
		"Central NACK should be forwarded to Compliance": {
			centralReply:      central.NodeInventoryACK_NACK,
			expectedACKCount:  0,
			expectedNACKCount: 1,
		},
	}

	for name, tc := range cases {
		ch <- fakeNodeInventory("node-" + name)
		s.NoError(mockCentralReply(h, tc.centralReply))
		result := consumeAndCountCompliance(h.ComplianceC(), 1)
		s.NoError(result.sc.Stopped().Wait())
		s.Equal(tc.expectedACKCount, result.ACKCount)
		s.Equal(tc.expectedNACKCount, result.NACKCount)
	}

	h.Stop(nil)
	s.T().Logf("waiting for handler to stop")
	s.NoError(h.Stopped().Wait())
}

// This test simulates a running Sensor loosing connection to Central, followed by a reconnect.
// As soon as Sensor enters offline mode, it should send NACKs to Compliance.
// In online mode, inventories are forwarded to Central, which responds with an ACK, that is passed to Compliance.
func (s *NodeInventoryHandlerTestSuite) TestHandlerOfflineACKNACK() {
	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	s.NoError(h.Start())

	states := []testState{
		{
			event:             common.SensorComponentEventCentralReachable,
			expectedACKCount:  1,
			expectedNACKCount: 0,
		},
		{
			event:             common.SensorComponentEventOfflineMode,
			expectedACKCount:  0,
			expectedNACKCount: 1,
		},
		{
			event:             common.SensorComponentEventCentralReachable,
			expectedACKCount:  1,
			expectedNACKCount: 0,
		},
	}

	for i, state := range states {
		h.Notify(state.event)
		ch <- fakeNodeInventory(fmt.Sprintf("Node-%d", i))
		if state.event == common.SensorComponentEventCentralReachable {
			s.NoError(mockCentralReply(h, central.NodeInventoryACK_ACK))
		}
		result := consumeAndCountCompliance(h.ComplianceC(), 1)
		s.NoError(result.sc.Stopped().Wait())
		s.Equal(state.expectedACKCount, result.ACKCount)
		s.Equal(state.expectedNACKCount, result.NACKCount)
	}

	h.Stop(nil)
	s.T().Logf("waiting for handler to stop")
	s.NoError(h.Stopped().Wait())
}

func mockCentralReply(h *nodeInventoryHandlerImpl, ackType central.NodeInventoryACK_Action) error {
	select {
	case <-h.ResponsesC():
		err := h.ProcessMessage(&central.MsgToSensor{
			Msg: &central.MsgToSensor_NodeInventoryAck{NodeInventoryAck: &central.NodeInventoryACK{
				ClusterId: "4",
				NodeName:  "4",
				Action:    ackType,
			}},
		})
		return err
	case <-time.After(5 * time.Second):
		return errors.New("ResponsesC msg didn't arrive after 5 seconds")
	}
}

// generateTestInputNoClose generates numToProduce messages of type NodeInventory.
// It returns a channel that must be closed by the caller.
func (s *NodeInventoryHandlerTestSuite) generateTestInputNoClose(numToProduce int) (chan *storage.NodeInventory, concurrency.StopperClient) {
	input := make(chan *storage.NodeInventory)
	st := concurrency.NewStopper()
	go func() {
		defer st.Flow().ReportStopped()
		for i := 0; i < numToProduce; i++ {
			select {
			case <-st.Flow().StopRequested():
				return
			case input <- fakeNodeInventory(fmt.Sprintf("Node-%d", i)):
			}
		}
	}()
	return input, st.Client()
}

// consumeAndCount consumes maximally numToConsume messages from the channel and counts the consumed messages
// It sets the Stopper in error state if the number of messages consumed were less than numToConsume.
func consumeAndCount[T any](ch <-chan T, numToConsume int) concurrency.StopperClient {
	st := concurrency.NewStopper()
	go func() {
		defer st.Flow().ReportStopped()
		for i := 0; i < numToConsume; i++ {
			select {
			case <-st.Flow().StopRequested():
				st.LowLevel().ResetStopRequest()
				st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
				return
			case _, ok := <-ch:
				if !ok {
					st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
					return
				}
			}
		}
	}()
	return st.Client()
}

type messageStats struct {
	NACKCount int
	ACKCount  int
	sc        concurrency.StopperClient
}

func consumeAndCountCompliance(ch <-chan common.MessageToComplianceWithAddress, numToConsume int) *messageStats {
	ms := &messageStats{0, 0, nil}
	st := concurrency.NewStopper()
	go func() {
		defer st.Flow().ReportStopped()
		for i := 0; i < numToConsume; i++ {
			select {
			case <-st.Flow().StopRequested():
				st.LowLevel().ResetStopRequest()
				st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
				return
			case msg, ok := <-ch:
				if !ok {
					st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
					return
				}
				switch msg.Msg.GetAck().GetAction() {
				case sensor.MsgToCompliance_NodeInventoryACK_ACK:
					ms.ACKCount++
				case sensor.MsgToCompliance_NodeInventoryACK_NACK:
					ms.NACKCount++
				}
			}
		}
	}()
	ms.sc = st.Client()
	return ms
}

func (s *NodeInventoryHandlerTestSuite) TestMultipleStartHandler() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})

	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	s.ErrorIs(h.Start(), errStartMoreThanOnce)

	consumer := consumeAndCount(h.ResponsesC(), 10)

	s.ErrorIs(h.Start(), errStartMoreThanOnce)

	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())

	// No second start even after a stop
	s.ErrorIs(h.Start(), errStartMoreThanOnce)
}

func (s *NodeInventoryHandlerTestSuite) TestDoubleStopHandler() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())
	h.Stop(nil)
	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
	// it should not block
	s.NoError(h.Stopped().Wait())
}

func (s *NodeInventoryHandlerTestSuite) TestInputChannelClosed() {
	ch, producer := s.generateTestInputNoClose(10)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	// By closing the channel ch, we mark that the producer finished writing all messages to ch
	close(ch)
	// The handler will stop as there are no more messages to handle
	s.ErrorIs(h.Stopped().Wait(), errInputChanClosed)
}

func (s *NodeInventoryHandlerTestSuite) generateNilTestInputNoClose(numToProduce int) (chan *storage.NodeInventory, concurrency.StopperClient) {
	input := make(chan *storage.NodeInventory)
	st := concurrency.NewStopper()
	go func() {
		defer st.Flow().ReportStopped()
		for i := 0; i < numToProduce; i++ {
			select {
			case <-st.Flow().StopRequested():
				return
			case input <- nil:
			}
		}
	}()
	return input, st.Client()
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerNilInput() {
	ch, producer := s.generateNilTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateNilTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 0)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerNodeUnknown() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockNeverHitNodeIDMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	// expect centralConsumer to get 0 messages - sensor should drop inventory when node is not found
	centralConsumer := consumeAndCount(h.ResponsesC(), 0)
	// expect complianceConsumer to get 10 NACK messages
	complianceConsumer := consumeAndCount(h.ComplianceC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(centralConsumer.Stopped().Wait())
	s.NoError(complianceConsumer.Stopped().Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
}

func (s *NodeInventoryHandlerTestSuite) TestHandlerCentralNotReady() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})
	s.NoError(h.Start())
	// expect centralConsumer to get 0 messages - sensor should NACK to compliance when the connection with central is not ready
	centralConsumer := consumeAndCount(h.ResponsesC(), 0)
	// expect complianceConsumer to get 10 NACK messages
	complianceConsumer := consumeAndCount(h.ComplianceC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(centralConsumer.Stopped().Wait())
	s.NoError(complianceConsumer.Stopped().Wait())

	h.Stop(nil)
	s.T().Logf("waiting for handler to stop")
	s.NoError(h.Stopped().Wait())
}

// mockAlwaysHitNodeIDMatcher always finds a node when GetNodeResource is called
type mockAlwaysHitNodeIDMatcher struct{}

// GetNodeID always finds a hardcoded ID "abc"
func (c *mockAlwaysHitNodeIDMatcher) GetNodeID(_ string) (string, error) {
	return "abc", nil
}

// mockNeverHitNodeIDMatcher simulates inability to find a node when GetNodeResource is called
type mockNeverHitNodeIDMatcher struct{}

// GetNodeID never finds a node and returns error
func (c *mockNeverHitNodeIDMatcher) GetNodeID(_ string) (string, error) {
	return "", errors.New("cannot find node")
}
