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

	cHandler commandHandlerImpl
	mockService *mocks.MockService
}

func (s *CommandHandlerTestSuite) SetupTest() {
	reachable := &atomic.Bool{}
	reachable.Store(true)
	s.mockService=mocks.NewMockService(gomock.NewController(s.T()))

	s.cHandler = commandHandlerImpl{
		service: s.mockService,

		commands: make(chan *central.ScrapeCommand),
		updates:  make(chan *message.ExpiringMessage),

		scrapeIDToState: make(map[string]*scrapeState),

		stopper: concurrency.NewStopper(),
		centralReachable: reachable,
	}
}

func (s *CommandHandlerTestSuite) TestCommandHandlerStops() {
	s.mockService.EXPECT().Output()
	err := s.cHandler.Start()
	s.Require().NoError(err)
	s.True(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventOfflineMode)
	s.False(s.cHandler.centralReachable.Load())
	s.cHandler.Notify(common.SensorComponentEventCentralReachable)
	time.Sleep(2000 * time.Millisecond)
	s.True(s.cHandler.centralReachable.Load())
}