package compliance

import (
	"context"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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
	// arrivalAtCentral generates a message when MsgFromSensor arrives to central
	// we need it to avoid using "<-time.After(50 * time.Millisecond)" to wait for the first message to arrive
	arrivalAtCentral chan struct{}
}

func (s *NodeScanHandlerTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.toCentral = make(chan *central.MsgFromSensor)
	s.arrivalAtCentral = make(chan struct{})
}

func (s *NodeScanHandlerTestSuite) TearDownTest() {

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

// startFakeCentral consumes all messages from s.toCentral and counts them
func (s *NodeScanHandlerTestSuite) startFakeCentral() {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.toCentral:
				go func() {
					s.arrivalAtCentral <- struct{}{}
				}()
			}
		}
	}(s.ctx)
}

// TestStopHandler goal is to stop handler while there are still some messages to process
// in the channel passed into NewNodeScanHandler.
// We expect that premature stop of the handler produces no race condition or goroutine leak.
// Exec with: go test -race -count=1 -v -run ^TestNodeScanHandler$ ./sensor/common/compliance
func (s *NodeScanHandlerTestSuite) TestStopHandler() {
	defer s.cancel()

	nodeScans := make(chan *storage.NodeScanV2)
	h := NewNodeScanHandler(nodeScans)
	s.startFakeCentral()
	s.startForwardingToCentral(h.ResponsesC())
	// this is a producer that stops the handler after producing the first message and then sends 3 more messages
	go func(ctx context.Context) {
		defer close(nodeScans)
		for i := 0; i < 4; i++ {
			select {
			case <-ctx.Done():
				return
			case nodeScans <- fakeNodeScanV2("Node"):
				if i == 0 {
					h.Stop(nil)
				}
			}
		}
	}(s.ctx)

	err := h.Start()
	assert.NoError(s.T(), err)
	<-h.Stopped().Done()
	assert.True(s.T(), h.Stopped().IsDone())

	// expect the first message to arrive to Central within 2s, poll every 0.5s
	assert.Eventuallyf(s.T(), func() bool {
		<-s.arrivalAtCentral
		return true
	}, 2*time.Second, 500*time.Millisecond, "central did not receive any message within a given deadline")
}
