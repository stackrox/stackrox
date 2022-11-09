package compliance

import (
	"context"
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
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

func (s *NodeScanHandlerTestSuite) startForwardingToCentral(c <-chan *central.MsgFromSensor) {
	go func() {
		defer close(s.toCentral)
		for {
			select {
			case <-s.ctx.Done():
				return
			case msg, ok := <-c:
				if !ok {
					return
				}
				s.toCentral <- msg
			}
		}
	}()
}

// TestStopHandler goal is to stop handler while there are still some messages to process
// in the channel passed into NewNodeScanHandler.
// We expect that premature stop of the handler produces no race condition or goroutine leak.
// Exec with: go test -race -count=1 -v -run ^TestNodeScanHandler$ ./sensor/common/compliance
func (s *NodeScanHandlerTestSuite) TestStopHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	h := NewNodeScanHandler(nodeScans)
	// consume all messages send toCentral
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.toCentral:
			}
		}
	}()
	s.startForwardingToCentral(h.ResponsesC())
	// this is a producer that stops the handler after producing the first message and then sends 3 more messages
	go func() {
		defer close(nodeScans)
		for i := 0; i < 4; i++ {
			select {
			case <-s.ctx.Done():
				return
			default:
				nodeScans <- fakeNodeScanV2("Node")
				if i == 0 {
					h.Stop(nil)
				}
			}
		}
	}()
	err := h.Start()
	assert.NoError(s.T(), err)
}

func (s *NodeScanHandlerTestSuite) TestRestartHandler() {
	nodeScans := make(chan *storage.NodeScanV2)
	defer close(nodeScans)
	h := NewNodeScanHandler(nodeScans)

	assert.NoError(s.T(), h.Start())
	h.Stop(nil)

	// try to start & stop the stopped handler again
	assert.Error(s.T(), h.Start())
	h.Stop(nil)

}
