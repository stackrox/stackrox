package output

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	waitTimeout = 100 * time.Millisecond
)

func shouldForwardMessage(t *testing.T, ch <-chan *message.ExpiringMessage) {
	select {
	case <-ch:
	case <-time.After(waitTimeout):
		t.Error("expecting event to have arrived")
	}
}

func shouldNotForwardMessage(t *testing.T, ch <-chan *message.ExpiringMessage) {
	select {
	case <-ch:
		t.Error("expecting event to not have arrived")
	case <-time.After(waitTimeout):
	}
}

func Test_OutputQueue_ExpiringMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	detector := mocks.NewMockDetector(ctrl)
	q := New(detector, 10)

	assert.NoError(t, q.Start())
	defer q.Stop(nil)

	expiredContext, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := map[string]struct {
		message   *component.ResourceEvent
		assertion func(*testing.T, <-chan *message.ExpiringMessage)
	}{
		"Empty context are treated as Background": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         nil,
			},
			assertion: shouldForwardMessage,
		},
		"Not expired message is sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         context.Background(),
			},
			assertion: shouldForwardMessage,
		},
		"Expired message is not sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         expiredContext,
			},
			assertion: shouldNotForwardMessage,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			detector.EXPECT().ReprocessDeployments(gomock.Eq([]string{}))
			q.Send(tc.message)
			tc.assertion(t, q.ResponsesC())
		})
	}
}
