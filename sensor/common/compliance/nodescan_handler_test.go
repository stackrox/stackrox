package compliance

import (
	"context"
	"errors"
	"fmt"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
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
	ctx, cancel := context.WithCancel(context.Background())
	suite.Run(t, &NodeScanHandlerTestSuite{
		ctx:    ctx,
		cancel: cancel,
	})
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
	cancel context.CancelFunc
	ctx    context.Context
}

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*loggingT).flushDaemon"),
	)
}

func (s *NodeScanHandlerTestSuite) TearDownTest() {
	defer assertNoGoroutineLeaks(s.T())
	s.cancel()
}

// stopAll gracefully stops stoppables
func stopAll(t *testing.T, stoppables ...stoppable) {
	for _, s := range stoppables {
		assert.NoError(t, s.signalAndWait(nil))
	}
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
// We expect that premature stop of the handler produces no race condition or goroutine leak.
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
	// This is to test whether the handler can shutdown with no leaks when the producer keeps producing messages.
	go func() {
		defer producer.signalStopped()
		for i := 0; i < 30; i++ {
			select {
			case <-producer.stopC.Done():
				return
			case nodeScans <- fakeNodeScanV2("Node"):
				if i == 0 {
					h.Stop(errTest)
				}
			}
		}
	}()
	s.NoError(consumer.stoppedC.Wait())
	s.ErrorIs(h.Stopped().Wait(), errTest)

	stopAll(s.T(), producer, consumer)
}

func (s *NodeScanHandlerTestSuite) TestHandlerRegularRoutine() {
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

	stopAll(s.T(), producer, consumer)
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

	stopAll(s.T(), producer, consumer)
}

// generateTestInputNoClose generates numToProduce messages of type NodeScanV2
// It returns channel that must be closed
func (s *NodeScanHandlerTestSuite) generateTestInputNoClose(numToProduce int) (chan *storage.NodeScanV2, stoppable) {
	input := make(chan *storage.NodeScanV2)
	st := newStoppable()
	// this is a producer that sends 10 nodescan messages
	go func() {
		defer st.signalStopped()
		for i := 0; i < numToProduce; i++ {
			select {
			case <-st.stopC.Done():
				return
			case input <- fakeNodeScanV2("Node"):
			}
		}
	}()
	return input, st
}

// consumeAndCount consumes maximally numToConsume messages from the channel and counts the consumed messages
// It sets error of stoppable.stopC if the number of messages consumed were less than numToConsume
func consumeAndCount[T any](ch <-chan *T, numToConsume int) stoppable {
	// simulate Central: consume all messages from h.ResponsesC()
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

func (s *NodeScanHandlerTestSuite) TestRestartHandler() {
	s.NotPanics(func() {
		ch, producer := s.generateTestInputNoClose(10)
		defer close(ch)
		h := NewNodeScanHandler(ch)
		s.NoError(h.Start())
		consumer := consumeAndCount(h.ResponsesC(), 10)
		s.NoError(producer.stoppedC.Wait())
		s.NoError(consumer.stoppedC.Wait())
		h.Stop(nil)

		err := h.Start()
		s.Error(err)
		s.ErrorIs(err, errStartMoreThanOnce)
		s.NoError(h.Stopped().Wait())

		stopAll(s.T(), producer, consumer)
	})
}

func (s *NodeScanHandlerTestSuite) TestDoubleStartHandler() {
	s.NotPanics(func() {
		ch, producer := s.generateTestInputNoClose(10)
		defer close(ch)
		h := NewNodeScanHandler(ch)
		s.NoError(h.Start())

		consumer := consumeAndCount(h.ResponsesC(), 10)
		s.NoError(producer.stoppedC.Wait())
		s.NoError(consumer.stoppedC.Wait())

		err := h.Start()
		s.Error(err)
		s.ErrorIs(err, errStartMoreThanOnce)
		h.Stop(nil)
		s.NoError(h.Stopped().Wait())

		stopAll(s.T(), producer, consumer)
	})
}

func (s *NodeScanHandlerTestSuite) TestDoubleStopHandler() {
	s.NotPanics(func() {
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

		stopAll(s.T(), producer, consumer)
	})
}
