package compliance

import (
	"context"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*loggingT).flushDaemon"),
	)
}
func TestNodeScanHandler(t *testing.T) {
	suite.Run(t, new(NodeScanHandlerTestSuite))
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

type NodeScanHandlerTestSuite struct {
	suite.Suite
	cancel    context.CancelFunc
	ctx       context.Context
	toCentral chan *central.MsgFromSensor
}

func (s *NodeScanHandlerTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.toCentral = make(chan *central.MsgFromSensor)
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
	h := NewNodeScanHandler(nodeScans)
	// simulate Central: consume all messages from h.ResponsesC()
	go func() {
		defer close(s.toCentral)
		for {
			select {
			case <-s.ctx.Done():
				return
			case _, ok := <-h.ResponsesC():
				if !ok {
					return
				}
			}
		}
	}()
	// this is a producer that stops the handler after producing the first message and then sends 30 more messages
	go func() {
		defer close(nodeScans)
		for i := 0; i < 30; i++ {
			select {
			case <-s.ctx.Done():
				return
			case nodeScans <- fakeNodeScanV2("Node"):
				if i == 0 {
					h.Stop(nil)
				}
			}
		}
	}()

	err := h.Start()
	s.Assert().NoError(err)
}

func (s *NodeScanHandlerTestSuite) TestRestartHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	s.Assert().NotPanics(func() {
		defer close(nodeScans)
		h := NewNodeScanHandler(nodeScans)
		s.Assert().NoError(h.Start())
		h.Stop(nil)
		// try to start & stop the stopped handler again and see error "stopped handlers cannot be restarted"
		s.Assert().Error(h.Start())
		h.Stop(nil)
	})
}

func (s *NodeScanHandlerTestSuite) TestDoubleStartHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	s.Assert().NotPanics(func() {
		defer close(nodeScans)
		h := NewNodeScanHandler(nodeScans)
		s.Assert().NoError(h.Start())
		s.Assert().NoError(h.Start())
		h.Stop(nil)
	})

}
