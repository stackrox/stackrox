package enhancement

import (
	"context"
	"testing"
	"time"

	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
		activeRequests: map[string]*enhancementSignal{"1": {msgArrived: concurrency.NewSignal()}},
		lock:           sync.Mutex{},
	}
	msg := &central.DeploymentEnhancementResponse{
		Msg: &central.DeploymentEnhancementMessage{
			Id:          "1",
			Deployments: nil,
		},
	}

	// Simulate a duplicate message. Broker mustn't crash or deadlock
	b.NotifyDeploymentReceived(msg)
	// NotifyDeploymentReceived should remove the ID after updating the msg
	s.NotContains(b.activeRequests, "1")
	b.NotifyDeploymentReceived(msg)
}

func (s *BrokerTestSuite) TestDeploymentReceivedWritesMessage() {
	es := &enhancementSignal{msgArrived: concurrency.NewSignal()}
	b := Broker{
		activeRequests: map[string]*enhancementSignal{"1": es},
		lock:           sync.Mutex{},
	}
	msg := &central.DeploymentEnhancementResponse{
		Msg: &central.DeploymentEnhancementMessage{
			Id:          "1",
			Deployments: []*storage.Deployment{{}, {}},
		},
	}

	b.NotifyDeploymentReceived(msg)
	s.Len(es.msg.GetMsg().GetDeployments(), 2)
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

	_, err := b.SendAndWaitForEnhancedDeployments(context.Background(), fakeSensorConn, deployments, 100*time.Millisecond)

	s.ErrorContains(err, "timed out waiting for augmented deployment", "Expected the function to time out, but it didn't")
}

func (s *BrokerTestSuite) TestSendAndWaitForAugmentedDeploymentsWritesToActiveRequests() {
	fakeSensorConn := connMocks.NewMockSensorConnection(gomock.NewController(s.T()))
	fakeSensorConn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).AnyTimes()
	b := NewBroker()
	deployments := make([]*storage.Deployment, 0)

	_, err := b.SendAndWaitForEnhancedDeployments(context.Background(), fakeSensorConn, deployments, 10*time.Millisecond)
	s.NoError(err)

	s.Len(b.activeRequests, 1)
}
