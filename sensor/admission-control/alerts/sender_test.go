package alerts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

func TestAlertSender(t *testing.T) {
	suite.Run(t, new(alertSenderSuite))
}

type alertSenderSuite struct {
	suite.Suite
	service *alertSenderImpl
	cl      *fakeAdmissionControlManagementServiceClient
}

type responseRequestPair struct {
	request  *sensor.AdmissionControlAlerts
	response *types.Empty
	err      error
}

func (s *alertSenderSuite) TestSendAlertsToSensor() {
	ctx := context.Background()
	err := errors.New("error")
	alerts := createAlertsMessage(2)
	cases := map[string]struct {
		numAlertsPerMessage  int
		responseRequestPairs []*responseRequestPair
	}{
		"Stage alerts on error and retry": {
			responseRequestPairs: []*responseRequestPair{
				{
					request:  createAlertsRequest(alerts),
					response: nil,
					err:      err,
				},
				{
					request:  createAlertsRequest(alerts),
					response: &types.Empty{},
					err:      nil,
				},
			},
		},
		"Send alerts no error": {
			responseRequestPairs: []*responseRequestPair{
				{
					request:  createAlertsRequest(alerts),
					response: &types.Empty{},
					err:      nil,
				},
			},
		},
	}
	for testName, c := range cases {
		s.Run(testName, func() {
			ch := make(chan []*storage.Alert)
			defer close(ch)
			s.createFakeAlertSender(ch)
			ctxWithCancel, cancelFunc := context.WithCancel(ctx)
			wg := &sync.WaitGroup{}
			s.cl.configureFakeClient(c.responseRequestPairs, wg)

			s.service.Start(ctxWithCancel)
			ch <- alerts
			wg.Wait()
			cancelFunc()
			<-ctxWithCancel.Done()
		})
	}
}

func createAlertsRequest(alerts []*storage.Alert) *sensor.AdmissionControlAlerts {
	return &sensor.AdmissionControlAlerts{
		AlertResults: []*central.AlertResults{
			{
				Alerts: alerts,
			},
		},
	}
}

func createAlertsMessage(numAlerts int) []*storage.Alert {
	ret := make([]*storage.Alert, numAlerts)
	for i := 0; i < numAlerts; i++ {
		ret[i] = &storage.Alert{
			Id: fmt.Sprintf("alert-%d", i),
		}
	}
	return ret
}

func (s *alertSenderSuite) createFakeAlertSender(alertC <-chan []*storage.Alert) {
	s.cl = &fakeAdmissionControlManagementServiceClient{
		t: s.T(),
	}
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 1 * time.Second
	eb.InitialInterval = 1 * time.Millisecond
	eb.MaxElapsedTime = 10 * time.Second
	eb.Reset()
	s.service = &alertSenderImpl{
		client:       s.cl,
		stagedAlerts: make(map[alertResultsIndicator]*central.AlertResults),
		alertsC:      alertC,
		stopC:        concurrency.NewSignal(),
		eb:           eb,
	}
}

type fakeAdmissionControlManagementServiceClient struct {
	sensor.AdmissionControlManagementServiceClient
	t                    *testing.T
	mu                   sync.Mutex
	responseRequestPairs []*responseRequestPair
	wg                   *sync.WaitGroup
	currResponse         int
}

func (c *fakeAdmissionControlManagementServiceClient) configureFakeClient(responseRequestPairs []*responseRequestPair, wg *sync.WaitGroup) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseRequestPairs = responseRequestPairs
	c.wg = wg
	c.wg.Add(len(c.responseRequestPairs))
}

func (c *fakeAdmissionControlManagementServiceClient) PolicyAlerts(_ context.Context, req *sensor.AdmissionControlAlerts, _ ...grpc.CallOption) (*types.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		c.currResponse++
		c.wg.Done()
	}()
	if c.currResponse >= len(c.responseRequestPairs) {
		c.t.Error("To many calls to PolicyAlerts")
	}
	if c.responseRequestPairs[c.currResponse].request != nil {
		assert.Equal(c.t, c.responseRequestPairs[c.currResponse].request, req)
	}
	return c.responseRequestPairs[c.currResponse].response, c.responseRequestPairs[c.currResponse].err
}
