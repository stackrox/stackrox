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
	dem := &central.DeploymentEnhancementMessage{}
	dem.SetId("1")
	dem.SetDeployments(nil)
	msg := &central.DeploymentEnhancementResponse{}
	msg.SetMsg(dem)

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
	msg := central.DeploymentEnhancementResponse_builder{
		Msg: central.DeploymentEnhancementMessage_builder{
			Id:          "1",
			Deployments: []*storage.Deployment{{}, {}},
		}.Build(),
	}.Build()

	b.NotifyDeploymentReceived(msg)
	s.Len(es.msg.GetMsg().GetDeployments(), 2)
}

func (s *BrokerTestSuite) TestSendAndWaitForEnhancedDeploymentsTimeout() {
	deployments := make([]*storage.Deployment, 0)
	fakeSensorConn := connMocks.NewMockSensorConnection(gomock.NewController(s.T()))
	fakeSensorConn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).AnyTimes()

	b := NewBroker()

	_, err := b.SendAndWaitForEnhancedDeployments(context.Background(), fakeSensorConn, deployments, 100*time.Millisecond)

	s.ErrorContains(err, "timed out waiting for enhanced deployment", "Expected the function to time out, but it didn't")
}

func (s *BrokerTestSuite) TestSendAndWaitForEnhancedDeploymentsWritesToActiveRequests() {
	fakeSensorConn := connMocks.NewMockSensorConnection(gomock.NewController(s.T()))
	fakeSensorConn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).AnyTimes()
	b := NewBroker()
	deployments := make([]*storage.Deployment, 0)

	_, err := b.SendAndWaitForEnhancedDeployments(context.Background(), fakeSensorConn, deployments, 10*time.Millisecond)
	s.ErrorContains(err, "timed out waiting for enhanced deployment")

	s.Len(b.activeRequests, 1)
}
