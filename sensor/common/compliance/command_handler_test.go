package compliance

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
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
	reachable := &atomic.Bool{}
	reachable.Store(false)
	s.mockService = mocks.NewMockService(gomock.NewController(s.T()))

	s.cHandler = commandHandlerImpl{
		service: s.mockService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *message.ExpiringMessage),

		scrapeIDToState: make(map[string]*scrapeState),

		stopper:          concurrency.NewStopper(),
		centralReachable: reachable,
	}
}

func (s *CommandHandlerTestSuite) StartScrape(scrapeId string) {
	scrapeCommand := central.ScrapeCommand{
		ScrapeId: scrapeId,
		Command: &central.ScrapeCommand_StartScrape{
			StartScrape: &central.StartScrape{
				Hostnames: []string{"192.168.0.1", "127.0.0.1"},
				Standards: []string{"NIST-800-53", "CIS-OCP"},
			},
		},
	}
	s.mockService.EXPECT().RunScrape(gomock.Any())
	s.cHandler.commands <- &scrapeCommand
}

func (s *CommandHandlerTestSuite) StartCommandHandler() {
	s.mockService.EXPECT().Output()
	s.False(s.cHandler.centralReachable.Load())
	err := s.cHandler.Start()
	s.Require().NoError(err)
}

func (s *CommandHandlerTestSuite) TestCommandHandlerStops() {
	s.StartCommandHandler()

	s.True(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventOfflineMode)
	s.False(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)
	time.Sleep(2000 * time.Millisecond)
	s.True(s.cHandler.centralReachable.Load())
}

func (s *CommandHandlerTestSuite) TestNoResponseWhenOffline() {
	s.StartCommandHandler()
	s.cHandler.Notify(common.SensorComponentEventOfflineMode)

	s.StartScrape("foo")
	s.mockService.EXPECT().Output()
	time.Sleep(time.Millisecond * 500)
	s.Empty(s.cHandler.updates)
}

func (s *CommandHandlerTestSuite) TestResponseWhenCentralReachable() {
	s.StartCommandHandler()
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)

	s.StartScrape("bar")
	time.Sleep(time.Millisecond * 500)
	select {
	case <-s.cHandler.updates:
		return
	case <-time.After(time.Second * 2):
		s.Fail("Timed out waiting for update")
	}
}
