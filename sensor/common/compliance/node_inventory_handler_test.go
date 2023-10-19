package compliance

import (
	"errors"
	"fmt"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
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
		goleak.IgnoreTopFunction("github.com/golang/glog.(*loggingT).flushDaemon"),
	)
}

func (s *NodeInventoryHandlerTestSuite) TearDownTest() {
	assertNoGoroutineLeaks(s.T())
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
func consumeAndCount[T any](ch <-chan *T, numToConsume int) concurrency.StopperClient {
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

func (s *NodeInventoryHandlerTestSuite) TestMultipleStartHandler() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeInventoryHandler(ch, &mockAlwaysHitNodeIDMatcher{})

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
	s.NoError(h.Start())
	// expect consumer to get 0 messages - sensor should drop inventory when node is not found
	consumer := consumeAndCount(h.ResponsesC(), 0)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
}

// mockAlwaysHitNodeIDMatcher always finds a node when GetNodeResource is called
type mockAlwaysHitNodeIDMatcher struct{}

// GetNodeID always finds a hardcoded ID "abc"
func (c *mockAlwaysHitNodeIDMatcher) GetNodeID(nodename string) (string, error) {
	return "abc", nil
}

// mockNeverHitNodeIDMatcher simulates inability to find a node when GetNodeResource is called
type mockNeverHitNodeIDMatcher struct{}

// GetNodeID never finds a node and returns error
func (c *mockNeverHitNodeIDMatcher) GetNodeID(nodename string) (string, error) {
	return "", errors.New("cannot find node")
}
