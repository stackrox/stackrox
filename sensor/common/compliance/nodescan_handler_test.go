package compliance

import (
	"context"
	"errors"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

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
					Module:    "FakeMod",
					Cpes:      []string{"cpe:/a:redhat:enterprise_linux:8::baseos"},
					AddedBy:   "FakeLayer",
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

// TestStopHandler goal is to stop handler while there are still some messages to process
// in the channel passed into NewNodeScanHandler.
// We expect that premature stop of the handler produces no race condition or goroutine leak.
// Exec with: go test -race -count=1 -v -run ^TestNodeScanHandler$ ./sensor/common/compliance
func (s *NodeScanHandlerTestSuite) TestStopHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	errTest := errors.New("example-stop-error")
	h := NewNodeScanHandler(nodeScans)
	s.consumeToCentral(h)
	// This is a producer that stops the handler after producing the first message and then sends many (29) more messages
	// The intent here it to test whether the handler can shutdown with no leaks when the context is canceled despite of
	// multiple messages waiting to be written to the channel
	go func() {
		defer close(nodeScans)
		for i := 0; i < 30; i++ {
			select {
			case <-s.ctx.Done():
				return
			case nodeScans <- fakeNodeScanV2("Node"):
				if i == 0 {
					h.Stop(errTest)
				}
			}
		}
	}()
	s.NoError(h.Start())
	err := h.Stopped().Wait()
	// if the goroutine finishes before h.Stopped() is cone, then err is errInputChanClosed
	// otherwise it is errTest. Both are fine as a reason for stopping the handler
	s.True(errors.Is(err, errTest) || errors.Is(err, errInputChanClosed))
}

func (s *NodeScanHandlerTestSuite) TestHandlerRegularRoutine() {
	h := NewNodeScanHandler(s.generateTestInput())
	s.NoError(h.Start())
	s.consumeToCentral(h)
	errTest := errors.New("example-stop-error")
	h.Stop(errTest)
	err := h.Stopped().Wait()
	s.True(errors.Is(err, errTest) || errors.Is(err, errInputChanClosed))
}

func (s *NodeScanHandlerTestSuite) TestHandlerStoppedError() {
	h := NewNodeScanHandler(s.generateTestInput())
	s.NoError(h.Start())
	s.consumeToCentral(h)
	errTest := errors.New("example-stop-error")
	h.Stop(errTest)
	err := h.Stopped().Wait()
	// if generateTestInput finishes before call to Stop(), err is errInputChanClosed
	// otherwise it is errTest. Both are fine as a reason for stopping the handler
	s.True(errors.Is(err, errTest) || errors.Is(err, errInputChanClosed))
}

// generateTestInput generates 10 input messages to the NodeScanHandler
func (s *NodeScanHandlerTestSuite) generateTestInput() <-chan *storage.NodeScanV2 {
	input := make(chan *storage.NodeScanV2)
	// this is a producer that sends 10 nodescan messages
	go func() {
		defer close(input)
		for i := 0; i < 10; i++ {
			select {
			case <-s.ctx.Done():
				return
			case input <- fakeNodeScanV2("Node"):
			}
		}
	}()
	return input
}

// consumeToCentral starts handler and blocks until it stops
func (s *NodeScanHandlerTestSuite) consumeToCentral(han NodeScanHandler) {
	// simulate Central: consume all messages from h.ResponsesC()
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case _, ok := <-han.ResponsesC():
				if !ok {
					return
				}
			}
		}
	}()
}

func (s *NodeScanHandlerTestSuite) TestRestartHandler() {
	s.NotPanics(func() {
		h := NewNodeScanHandler(s.generateTestInput())
		s.NoError(h.Start())
		s.consumeToCentral(h)
		h.Stop(nil)

		err := h.Start()
		s.Error(err)
		s.ErrorIs(err, errStartMoreThanOnce)
	})
}

func (s *NodeScanHandlerTestSuite) TestDoubleStartHandler() {
	s.NotPanics(func() {
		h := NewNodeScanHandler(s.generateTestInput())
		s.NoError(h.Start())
		s.consumeToCentral(h)

		err := h.Start()
		s.Error(err)
		s.ErrorIs(err, errStartMoreThanOnce)
		h.Stop(nil)
	})
}

func (s *NodeScanHandlerTestSuite) TestDoubleStopHandler() {
	s.NotPanics(func() {
		h := NewNodeScanHandler(s.generateTestInput())
		s.NoError(h.Start())
		s.consumeToCentral(h)
		h.Stop(nil)
		h.Stop(nil)
	})
}
