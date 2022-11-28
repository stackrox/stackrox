package compliance

import (
	"errors"
	"fmt"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

// stoppable represents a gracefully stoppable thing, e.g., async process
type stoppable struct {
	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func newStoppable() stoppable {
	return stoppable{
		stoppedC: concurrency.NewErrorSignal(),
		stopC:    concurrency.NewErrorSignal(),
	}
}

// signalStopped marks the stoppaple thing as stopped
func (st *stoppable) signalStopped() {
	st.stoppedC.SignalWithError(st.stopC.Err())
}

// signalAndWait sends a command to stop the stoppaple thing with an error and waits until it stops
func (st *stoppable) signalAndWait(err error) error {
	st.stopC.SignalWithError(err)
	return st.stoppedC.Wait()
}

func TestNodeScanHandler(t *testing.T) {
	suite.Run(t, &NodeScanHandlerTestSuite{})
}

func fakeNodeScanV2(nodeName string) *storage.NodeScanV2 {
	msg := &storage.NodeScanV2{
		NodeId:   "",
		NodeName: nodeName,
		ScanTime: timestamp.TimestampNow(),
		Components: &scannerV1.Components{
			Namespace: "Testme OS",
			RhelComponents: []*scannerV1.RHELComponent{
				{
					Id:        int64(1),
					Name:      "vim-minimal",
					Namespace: "rhel:8",
					Version:   "2:7.4.629-6.el8",
					Arch:      "x86_64",
					Module:    "",
					Cpes:      []string{"cpe:/a:redhat:enterprise_linux:8::baseos"},
					AddedBy:   "hardcoded",
				},
			},
			LanguageComponents: nil,
		},
		Notes: nil,
	}
	return msg
}

var _ suite.TearDownTestSuite = (*NodeScanHandlerTestSuite)(nil)

type NodeScanHandlerTestSuite struct {
	suite.Suite
}

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*loggingT).flushDaemon"),
	)
}

func (s *NodeScanHandlerTestSuite) TearDownTest() {
	assertNoGoroutineLeaks(s.T())
}

func (s *NodeScanHandlerTestSuite) TestResponsesCShouldPanicWhenNotStarted() {
	nodeScans := make(chan *storage.NodeScanV2)
	defer close(nodeScans)
	h := NewNodeScanHandler(nodeScans)
	s.Panics(func() {
		h.ResponsesC()
	})
}

// TestStopHandler goal is to stop handler while there are still some messages to process
// in the channel passed into NewNodeScanHandler.
// We expect that premature stop of the handler results in a clean stop without any race conditions or goroutine leaks.
// Exec with: go test -race -count=1 -v -run ^TestNodeScanHandler$ ./sensor/common/compliance
func (s *NodeScanHandlerTestSuite) TestStopHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	defer close(nodeScans)
	producer := newStoppable()
	errTest := errors.New("example-stop-error")
	h := NewNodeScanHandler(nodeScans)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 1)
	// This is a producer that stops the handler after producing the first message and then sends many (29) more messages.
	go func() {
		defer producer.signalStopped()
		for i := 0; i < 30; i++ {
			select {
			case <-producer.stopC.Done():
				return
			case nodeScans <- fakeNodeScanV2("Node"):
				if i == 0 {
					s.NoError(consumer.stoppedC.Wait()) // This blocks until consumer receives its 1 message
					h.Stop(errTest)
				}
			}
		}
	}()
	s.ErrorIs(h.Stopped().Wait(), errTest)
	s.NoError(producer.signalAndWait(nil))
}

func (s *NodeScanHandlerTestSuite) TestHandlerRegularRoutine() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeScanHandler(ch)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.stoppedC.Wait())
	s.NoError(consumer.stoppedC.Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
}

func (s *NodeScanHandlerTestSuite) TestHandlerStoppedError() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeScanHandler(ch)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.stoppedC.Wait())
	s.NoError(consumer.stoppedC.Wait())

	errTest := errors.New("example-stop-error")
	h.Stop(errTest)
	s.ErrorIs(h.Stopped().Wait(), errTest)
}

// generateTestInputNoClose generates numToProduce messages of type NodeScanV2.
// It returns a channel that must be closed by the caller.
func (s *NodeScanHandlerTestSuite) generateTestInputNoClose(numToProduce int) (chan *storage.NodeScanV2, stoppable) {
	input := make(chan *storage.NodeScanV2)
	st := newStoppable()
	go func() {
		defer st.signalStopped()
		for i := 0; i < numToProduce; i++ {
			select {
			case <-st.stopC.Done():
				return
			case input <- fakeNodeScanV2(fmt.Sprintf("Node-%d", i)):
			}
		}
	}()
	return input, st
}

// consumeAndCount consumes maximally numToConsume messages from the channel and counts the consumed messages
// It sets error of stoppable.stopC if the number of messages consumed were less than numToConsume
func consumeAndCount[T any](ch <-chan *T, numToConsume int) stoppable {
	st := newStoppable()
	go func() {
		defer st.signalStopped()
		for i := 0; i < numToConsume; i++ {
			select {
			case <-st.stopC.Done():
				st.stopC.Reset()
				st.stopC.SignalWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
				return
			case _, ok := <-ch:
				if !ok {
					st.stopC.SignalWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
					return
				}
			}
		}
	}()
	return st
}

func (s *NodeScanHandlerTestSuite) TestMultipleStartHandler() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeScanHandler(ch)

	s.NoError(h.Start())
	s.ErrorIs(h.Start(), errStartMoreThanOnce)

	consumer := consumeAndCount(h.ResponsesC(), 10)

	s.ErrorIs(h.Start(), errStartMoreThanOnce)

	s.NoError(producer.stoppedC.Wait())
	s.NoError(consumer.stoppedC.Wait())

	h.Stop(nil)
	s.NoError(h.Stopped().Wait())

	// No second start even after a stop
	s.ErrorIs(h.Start(), errStartMoreThanOnce)
}

func (s *NodeScanHandlerTestSuite) TestDoubleStopHandler() {
	ch, producer := s.generateTestInputNoClose(10)
	defer close(ch)
	h := NewNodeScanHandler(ch)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.stoppedC.Wait())
	s.NoError(consumer.stoppedC.Wait())
	h.Stop(nil)
	h.Stop(nil)
	s.NoError(h.Stopped().Wait())
	// it should not block
	s.NoError(h.Stopped().Wait())
}

func (s *NodeScanHandlerTestSuite) TestInputChannelClosed() {
	ch, producer := s.generateTestInputNoClose(10)
	h := NewNodeScanHandler(ch)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.stoppedC.Wait())
	s.NoError(consumer.stoppedC.Wait())

	// By closing the channel ch, we mark that the producer finished writing all messages to ch
	close(ch)
	// The handler will stop as there are no more messages to handle
	s.ErrorIs(h.Stopped().Wait(), errInputChanClosed)
}
