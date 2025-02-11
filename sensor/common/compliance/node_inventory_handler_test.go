package compliance

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
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
		ScanTime: protocompat.TimestampNow(),
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

func fakeNodeIndex(arch string) *v4.IndexReport {
	return &v4.IndexReport{
		HashId:  fmt.Sprintf("sha256:%s", strings.Repeat("a", 64)),
		Success: true,
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				exemplaryPackage("0", "vim-minimal", arch),
				exemplaryPackage("1", "vim-minimal-noarch", "noarch"),
				exemplaryPackage("2", "vim-minimal-empty-arch", ""),
			},
			Repositories: []*v4.Repository{
				exemplaryRepo("0"),
				exemplaryRepo("1"),
				exemplaryRepo("2"),
			},
		},
	}
}

func exemplaryPackage(id, name, arch string) *v4.Package {
	return &v4.Package{
		Id:      id,
		Name:    name,
		Version: "2:7.4.629-6.el8",
		Kind:    "binary",
		Source: &v4.Package{
			Name:    "vim",
			Version: "2:7.4.629-6.el8",
			Kind:    "source",
			Source:  nil,
			Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
		},
		PackageDb:      "sqlite:usr/share/rpm",
		RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
		Arch:           arch,
		Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
	}
}

func exemplaryRepo(id string) *v4.Repository {
	return &v4.Repository{
		Id:   id,
		Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
		Key:  "rhel-cpe-repository",
		Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
	}
}

var _ suite.TearDownTestSuite = (*NodeInventoryHandlerTestSuite)(nil)

type NodeInventoryHandlerTestSuite struct {
	suite.Suite
}

func assertNoGoroutineLeaks(t *testing.T) {
	goleak.VerifyNone(t,
		// Ignore a known leak: https://github.com/DataDog/dd-trace-go/issues/1469
		goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
		// Ignore a known leak caused by importing the GCP cscc SDK.
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
	)
}

func (s *NodeInventoryHandlerTestSuite) TearDownTest() {
	assertNoGoroutineLeaks(s.T())
}

func (s *NodeInventoryHandlerTestSuite) TestExtractArch() {
	cases := map[string]struct {
		rpmArch      string
		expectedArch string
	}{
		"noarch": {
			rpmArch:      "noarch",
			expectedArch: "",
		},
		"empty-arch": {
			rpmArch:      "",
			expectedArch: "",
		},
		"x86_64": {
			rpmArch:      "x86_64",
			expectedArch: "x86_64",
		},
		"foobar": {
			rpmArch:      "foobar",
			expectedArch: "foobar",
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			got := extractArch(fakeNodeIndex(tc.rpmArch))
			s.Equal(tc.expectedArch, got)
		})
	}
}

func (s *NodeInventoryHandlerTestSuite) TestAttachRPMtoRHCOS() {
	arch := "x86_64"
	rpmIR := fakeNodeIndex(arch)
	got := attachRPMtoRHCOS("417.94.202501071621-0", arch, rpmIR)

	s.Lenf(got.GetContents().GetPackages(), len(rpmIR.GetContents().GetPackages())+1, "IR should have 1 extra package")
	s.Lenf(got.GetContents().GetEnvironments(), len(rpmIR.GetContents().GetEnvironments())+1, "IR should have 1 extra envinronment")
	s.Lenf(got.GetContents().GetRepositories(), len(rpmIR.GetContents().GetRepositories())+1, "IR should have 1 extra repository")

	var rhcosPKG *v4.Package
	for _, p := range got.GetContents().GetPackages() {
		if p.GetName() == "rhcos" {
			rhcosPKG = p
			break
		}
	}
	s.Require().NotNil(rhcosPKG, "the 'rhcos' pkg should exist in node index")
	s.Equal("rhcos", rhcosPKG.GetName())
	s.Equal(arch, rhcosPKG.GetArch())
	s.Equal("600", rhcosPKG.GetId())

	var rhcosRepo *v4.Repository
	for _, r := range got.GetContents().GetRepositories() {
		if r.GetId() == "600" {
			rhcosRepo = r
			break
		}
	}
	s.Require().NotNil(rhcosRepo, "the golden repos should exist in node index")
	s.Equal("", rhcosRepo.GetKey())
	s.Equal(goldenName, rhcosRepo.GetName())
	s.Equal(goldenURI, rhcosRepo.GetUri())
}

func (s *NodeInventoryHandlerTestSuite) TestCapabilities() {
	inventories := make(chan *storage.NodeInventory)
	defer close(inventories)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(inventories, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
	s.Nil(h.Capabilities())
}

func (s *NodeInventoryHandlerTestSuite) TestResponsesCShouldPanicWhenNotStarted() {
	inventories := make(chan *storage.NodeInventory)
	defer close(inventories)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(inventories, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	producer := concurrency.NewStopper()
	h := NewNodeInventoryHandler(inventories, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	cases := map[string]struct {
		centralReplies    []central.NodeInventoryACK_Action
		expectedACKCount  int
		expectedNACKCount int
	}{
		"Central ACK should be forwarded to Compliance": {
			centralReplies:    []central.NodeInventoryACK_Action{central.NodeInventoryACK_ACK},
			expectedACKCount:  1,
			expectedNACKCount: 0,
		},
		"Central NACK should be forwarded to Compliance": {
			centralReplies:    []central.NodeInventoryACK_Action{central.NodeInventoryACK_NACK},
			expectedACKCount:  0,
			expectedNACKCount: 1,
		},
		"Multiple ACK messages should be forwarded to Compliance": {
			centralReplies: []central.NodeInventoryACK_Action{
				central.NodeInventoryACK_ACK, central.NodeInventoryACK_ACK,
				central.NodeInventoryACK_NACK, central.NodeInventoryACK_NACK,
			},
			expectedACKCount:  2,
			expectedNACKCount: 2,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			ch := make(chan *storage.NodeInventory)
			defer close(ch)
			reports := make(chan *index.IndexReportWrap)
			defer close(reports)
			handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
			s.NoError(handler.Start())
			handler.Notify(common.SensorComponentEventCentralReachable)

			go func() {
				for i := 0; i < len(tc.centralReplies); i++ {
					ch <- fakeNodeInventory(fmt.Sprintf("node-%s-%d", name, i))
				}
			}()

			result := consumeAndCountCompliance(s.T(), handler.ComplianceC(), tc.expectedACKCount+tc.expectedNACKCount)

			for _, reply := range tc.centralReplies {
				s.NoError(mockCentralReply(handler, reply))
			}

			s.NoError(result.sc.Stopped().Wait())
			s.Equal(tc.expectedACKCount, result.ACKCount)
			s.Equal(tc.expectedNACKCount, result.NACKCount)

			handler.Stop(nil)
			s.T().Logf("waiting for handler to stop")
			s.NoError(handler.Stopped().Wait())
		})

	}

}

// This test simulates a running Sensor loosing connection to Central, followed by a reconnect.
// As soon as Sensor enters offline mode, it should send NACKs to Compliance.
// In online mode, inventories are forwarded to Central, which responds with an ACK, that is passed to Compliance.
func (s *NodeInventoryHandlerTestSuite) TestHandlerOfflineACKNACK() {
	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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

		result := consumeAndCountCompliance(s.T(), h.ComplianceC(), state.expectedACKCount+state.expectedNACKCount)

		if state.event == common.SensorComponentEventCentralReachable {
			s.NoError(mockCentralReply(h, central.NodeInventoryACK_ACK))
		}
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
		return h.ProcessMessage(&central.MsgToSensor{
			Msg: &central.MsgToSensor_NodeInventoryAck{NodeInventoryAck: &central.NodeInventoryACK{
				ClusterId: "4",
				NodeName:  "4",
				Action:    ackType,
			}},
		})
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

func consumeAndCountCompliance(t *testing.T, ch <-chan common.MessageToComplianceWithAddress, numToConsume int) *messageStats {
	ms := &messageStats{0, 0, nil}
	st := concurrency.NewStopper()
	go func() {
		defer st.Flow().ReportStopped()
		for i := 0; i < numToConsume; i++ {
			select {
			case <-st.Flow().StopRequested():
				t.Logf("Stop requested")
				st.LowLevel().ResetStopRequest()
				st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
				return
			case msg, ok := <-ch:
				t.Logf("Got message: %v", msg)
				if !ok {
					t.Logf("CH channel closed")
					st.Flow().StopWithError(fmt.Errorf("consumer consumed %d messages but expected to do %d", i, numToConsume))
					return
				}
				t.Logf("Executing ++ on action %s", msg.Msg.GetAck().GetAction())
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})

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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
	// Notify is called before Start to avoid race between generateTestInputNoClose and the NodeInventoryHandler
	h.Notify(common.SensorComponentEventCentralReachable)
	s.NoError(h.Start())
	consumer := consumeAndCount(h.ResponsesC(), 10)
	s.NoError(producer.Stopped().Wait())
	s.NoError(consumer.Stopped().Wait())

	// By closing the channel ch, we mark that the producer finished writing all messages to ch
	close(ch)
	// The handler will stop as there are no more messages to handle
	s.ErrorIs(h.Stopped().Wait(), errInventoryInputChanClosed)
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockNeverHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	h := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{})
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

// mockNeverHitNodeIDMatcher simulates inability to find a node when GetNodeResource is called
type mockRHCOSNodeMatcher struct{}

// GetRHCOSVersion always identifies as RHCOS and provides a valid version
func (c *mockRHCOSNodeMatcher) GetRHCOSVersion(_ string) (bool, string, error) {
	return true, "417.94.202412120651-0", nil
}
