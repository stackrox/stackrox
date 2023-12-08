package augmentation

import (
	"context"
	"testing"
	"time"

	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBroker(t *testing.T) {
	suite.Run(t, &BrokerTestSuite{})
}

type BrokerTestSuite struct {
	suite.Suite
}

func (s *BrokerTestSuite) TestNotifyDeploymentReceivedDoubleMessage() {
	b := Broker{
		requests: map[string]chan<- *central.DeploymentEnhancementResponse{"1": make(chan<- *central.DeploymentEnhancementResponse, 2)},
		lock:     sync.Mutex{},
	}
	msg := &central.DeploymentEnhancementResponse{
		Msg: &central.DeploymentEnhancementMessage{
			Id:          "1",
			Deployments: nil,
		},
	}

	// Simulate a duplicate message. Broker mustn't crash or deadlock
	b.NotifyDeploymentReceived(msg)
	b.NotifyDeploymentReceived(msg)
}

func (s *BrokerTestSuite) TestNotifyDeploymentReceivedMatchesID() {
	wg := sync.WaitGroup{}
	b := NewBroker()
	msg := &central.DeploymentEnhancementResponse{
		Msg: &central.DeploymentEnhancementMessage{
			Id:          "1",
			Deployments: nil,
		},
	}
	wg.Add(1)
	go func() {
		c := make(chan *central.DeploymentEnhancementResponse, 1)
		b.requests["1"] = c
		wg.Done()

		select {
		case <-time.After(2 * time.Second):
			s.Fail("did not receive response in time")
		case <-c:
		}
	}()
	wg.Wait()

	b.NotifyDeploymentReceived(msg)
}

func (s *BrokerTestSuite) TestSendAndWaitForAugmentedDeploymentsTimeout() {
	deployments := make([]*storage.Deployment, 0)
	fakeSensorConn := connMocks.NewMockSensorConnection(gomock.NewController(s.T()))
	fakeSensorConn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Do(
		func(c context.Context, msg *central.MsgToSensor) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		})
	b := NewBroker()

	_, err := b.SendAndWaitForAugmentedDeployments(context.Background(), fakeSensorConn, deployments, 100*time.Millisecond)

	s.ErrorContains(err, "timed out waiting for augmented deployment", "Expected the function to time out, but it didn't")
}
