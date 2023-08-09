package compliance

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCommandHandler(t *testing.T) {
	suite.Run(t, new(CommandHandlerTestSuite))
}

type CommandHandlerTestSuite struct {
	suite.Suite

	cHandler    commandHandlerImpl
	mockService *mocks.MockService
}

func (s *CommandHandlerTestSuite) SetupTest() {
	s.mockService = mocks.NewMockService(gomock.NewController(s.T()))

	s.cHandler = commandHandlerImpl{
		service: s.mockService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *message.ExpiringMessage),

		scrapeIDToState: make(map[string]*scrapeState),

		stopper:          concurrency.NewStopper(),
		centralReachable: atomic.Bool{},
	}
}

func (s *CommandHandlerTestSuite) startScrape(scrapeID string, hostnames []string) {
	scrapeCommand := central.ScrapeCommand{
		ScrapeId: scrapeID,
		Command: &central.ScrapeCommand_StartScrape{
			StartScrape: &central.StartScrape{
				Hostnames: hostnames,
				Standards: []string{"NIST-800-53", "CIS-OCP"},
			},
		},
	}
	s.mockService.EXPECT().RunScrape(gomock.Any()).DoAndReturn(func(_ any) int { return len(hostnames) })
	s.cHandler.commands <- &scrapeCommand
}

func (s *CommandHandlerTestSuite) getScrapeUpdate() {
	select {
	case update := <-s.cHandler.updates:
		message := update.GetMsg()
		scrapeUpdate, ok := message.(*central.MsgFromSensor_ScrapeUpdate)
		s.Require().True(ok)
		s.Assert().Equal("foo", scrapeUpdate.ScrapeUpdate.ScrapeId)
	case <-time.After(time.Second * 2):
		s.Require().Fail("Timed out waiting for update")
	}
}

func (s *CommandHandlerTestSuite) sendComplianceReturn(outputChan chan *compliance.ComplianceReturn) {
	outputChan <- &compliance.ComplianceReturn{
		NodeName:             "node1",
		ScrapeId:             "foo",
		DockerData:           nil,
		CommandLines:         nil,
		Files:                nil,
		SystemdFiles:         nil,
		ContainerRuntimeInfo: nil,
		Time:                 nil,
		Evidence:             nil,
	}
}

func (s *CommandHandlerTestSuite) startCommandHandler(outputChan chan *compliance.ComplianceReturn) {
	s.mockService.EXPECT().Output().AnyTimes().DoAndReturn(func() chan *compliance.ComplianceReturn { return outputChan })
	s.Assert().False(s.cHandler.centralReachable.Load())
	err := s.cHandler.Start()
	s.Require().NoError(err)
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)
}

func (s *CommandHandlerTestSuite) TestCommandHandlerStops() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)

	s.Assert().True(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventOfflineMode)
	s.Assert().False(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)
	time.Sleep(2000 * time.Millisecond)
	s.Assert().True(s.cHandler.centralReachable.Load())
}

func (s *CommandHandlerTestSuite) TestNoResponseWhenOffline() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)
	s.cHandler.Notify(common.SensorComponentEventOfflineMode)

	s.startScrape("foo", []string{"node1", "node2"})
	time.Sleep(time.Millisecond * 500)
	s.Assert().Empty(s.cHandler.updates)
}

func (s *CommandHandlerTestSuite) TestResponseWhenCentralReachable() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)

	s.startScrape("bar", []string{"node1", "node2"})
	select {
	case <-s.cHandler.updates:
		return
	case <-time.After(time.Second * 2):
		s.Require().Fail("Timed out waiting for update")
	}
}

func (s *CommandHandlerTestSuite) TestStartScrapeSucceedsComplianceMessageNotSent() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)

	s.startScrape("foo", []string{"node1", "node2"})

	s.getScrapeUpdate()

	s.cHandler.Notify(common.SensorComponentEventOfflineMode)

	s.sendComplianceReturn(outputChan)

	time.Sleep(time.Millisecond * 500)
	scrapeState, ok := s.cHandler.scrapeIDToState["foo"]
	s.Require().True(ok)
	s.Assert().False(scrapeState.remainingNodes.Contains("node1"))
	s.Assert().True(scrapeState.remainingNodes.Contains("node2"))
	s.Empty(s.cHandler.updates)
}

func (s *CommandHandlerTestSuite) TestStartScrapeSucceedsComplianceMessageSent() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)

	s.startScrape("foo", []string{"node1", "node2"})

	s.getScrapeUpdate()

	s.sendComplianceReturn(outputChan)

	select {
	case <-s.cHandler.updates:
		return
	case <-time.After(time.Second * 2):
		s.Require().Fail("Timed out waiting for update")
	}

	scrapeState, ok := s.cHandler.scrapeIDToState["foo"]
	s.Require().True(ok)
	s.Assert().False(scrapeState.remainingNodes.Contains("node1"))
	s.Assert().True(scrapeState.remainingNodes.Contains("node2"))
}
