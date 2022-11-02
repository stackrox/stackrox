package compliance

import (
	"context"
	"fmt"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
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
					Name:      "vim-minimal",
					Namespace: "rhel:8",
					Version:   "2:7.4.629-6.el8.x86_64",
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

type nodeScansProducerThatStops struct {
	numMsgs   int
	nodeScans chan *storage.NodeScanV2
}

type stoppable interface {
	Stop(error)
}

func newNodeScansProducerThatStops(num int) *nodeScansProducerThatStops {
	return &nodeScansProducerThatStops{
		numMsgs:   num,
		nodeScans: make(chan *storage.NodeScanV2),
	}
}

func (nsp *nodeScansProducerThatStops) generator() <-chan *storage.NodeScanV2 {
	return nsp.nodeScans
}

func (nsp *nodeScansProducerThatStops) start(s stoppable) {
	go func(numMsgs int) {
		defer close(nsp.nodeScans)
		for i := 0; i < numMsgs; i++ {
			nsp.nodeScans <- fakeNodeScanV2(fmt.Sprintf("Bobby_%d", i))
			if i == 0 {
				s.Stop(nil)
			}
		}
	}(nsp.numMsgs)
}

type NodeScanHandlerTestSuite struct {
	suite.Suite
	mu                   *sync.Mutex
	numReceivedAtCentral int
	cancel               context.CancelFunc
	ctx                  context.Context
	toCentral            chan *central.MsgFromSensor
}

func (s *NodeScanHandlerTestSuite) SetupTest() {
	s.mu = &sync.Mutex{}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.numReceivedAtCentral = 0
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.toCentral = make(chan *central.MsgFromSensor)
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
				s.mu.Lock()
				s.numReceivedAtCentral++
				s.mu.Unlock()
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
	numScans := 1000
	// nodeScansProducerThatStops stops the handler after producing the first message.
	p := newNodeScansProducerThatStops(numScans)
	h := NewNodeScanHandler(p.generator())
	p.start(h)
	s.startFakeCentral()
	s.startForwardingToCentral(h.ResponsesC())

	err := h.Start()
	assert.NoError(s.T(), err)
	<-h.Stopped().Done()
	assert.True(s.T(), h.Stopped().IsDone())

	// give central a chance to receive something before we lock s.mu
	<-time.After(50 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.LessOrEqual(s.T(), s.numReceivedAtCentral, numScans)
	assert.GreaterOrEqual(s.T(), s.numReceivedAtCentral, 1, "Handler should handle at least 1 msg")
	s.T().Logf("Handler managed to process %d msgs ouf of %d", s.numReceivedAtCentral, numScans)
}
