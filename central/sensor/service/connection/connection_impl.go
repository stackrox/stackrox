package connection

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/metadata"
)

var (
	log = logging.LoggerForModule()
)

type sensorConnection struct {
	clusterID           string
	stopSig, stoppedSig concurrency.ErrorSignal

	sendC chan *central.MsgToSensor

	scrapeCtrl          scrape.Controller
	networkPoliciesCtrl networkpolicies.Controller

	eventQueue    *dedupingQueue
	eventPipeline pipeline.ClusterPipeline

	checkInRecorder          CheckInRecorder
	checkInRecordRateLimiter *rate.Limiter
}

func newConnection(clusterID string, pf pipeline.Factory, recorder CheckInRecorder) (*sensorConnection, error) {
	eventPipeline, err := pf.PipelineForCluster(clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "creating event pipeline")
	}

	conn := &sensorConnection{
		stopSig:       concurrency.NewErrorSignal(),
		stoppedSig:    concurrency.NewErrorSignal(),
		sendC:         make(chan *central.MsgToSensor),
		eventPipeline: eventPipeline,
		eventQueue:    newDedupingQueue(),

		clusterID:       clusterID,
		checkInRecorder: recorder,

		checkInRecordRateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 1),
	}

	conn.scrapeCtrl = scrape.NewController(conn, &conn.stopSig)
	conn.networkPoliciesCtrl = networkpolicies.NewController(conn, &conn.stopSig)
	return conn, nil
}

func (c *sensorConnection) Terminate(err error) bool {
	return c.stopSig.SignalWithError(err)
}

func (c *sensorConnection) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedSig
}

// Record the check-in if the rate limiter allows it.
func (c *sensorConnection) recordCheckInRateLimited() {
	if c.checkInRecordRateLimiter.Allow() {
		err := c.checkInRecorder.UpdateClusterContactTime(context.TODO(), c.clusterID, time.Now())
		if err != nil {
			log.Warnf("Could not record cluster contact: %v", err)
		}
	}
}

func (c *sensorConnection) runRecv(server central.SensorService_CommunicateServer) {
	for !c.stopSig.IsDone() {
		msg, err := server.Recv()
		if err != nil {
			c.stopSig.SignalWithError(errors.Wrap(err, "recv error"))
			return
		}
		c.recordCheckInRateLimited()
		c.eventQueue.push(msg)
	}
}

func (c *sensorConnection) handleMessages() {
	for msg := c.eventQueue.pullBlocking(&c.stopSig); msg != nil; msg = c.eventQueue.pullBlocking(&c.stopSig) {
		if err := c.handleMessage(msg); err != nil {
			log.Errorf("Error handling sensor message: %v", err)
		}
	}
	c.eventPipeline.OnFinish(c.clusterID)
	c.stoppedSig.SignalWithError(c.stopSig.Err())
}

func (c *sensorConnection) runSend(server central.SensorService_CommunicateServer) {
	for !c.stopSig.IsDone() {
		select {
		case <-c.stopSig.Done():
			return
		case <-server.Context().Done():
			c.stopSig.SignalWithError(errors.Wrap(server.Context().Err(), "context error"))
			return
		case msg := <-c.sendC:
			if err := server.Send(msg); err != nil {
				c.stopSig.SignalWithError(errors.Wrap(err, "send error"))
				return
			}
		}
	}
}

func (c *sensorConnection) Scrapes() scrape.Controller {
	return c.scrapeCtrl
}

func (c *sensorConnection) InjectMessageIntoQueue(msg *central.MsgFromSensor) {
	c.eventQueue.push(msg)
}

func (c *sensorConnection) NetworkPolicies() networkpolicies.Controller {
	return c.networkPoliciesCtrl
}

func (c *sensorConnection) InjectMessage(ctx concurrency.Waitable, msg *central.MsgToSensor) error {
	select {
	case c.sendC <- msg:
		return nil
	case <-ctx.Done():
		return errors.New("context aborted")
	case <-c.stopSig.Done():
		return errors.Wrap(c.stopSig.Err(), "could not send message as sensor connection was stopped")
	}
}

func (c *sensorConnection) handleMessage(msg *central.MsgFromSensor) error {
	switch m := msg.Msg.(type) {
	case *central.MsgFromSensor_ScrapeUpdate:
		return c.scrapeCtrl.ProcessScrapeUpdate(m.ScrapeUpdate)
	case *central.MsgFromSensor_NetworkPoliciesResponse:
		return c.networkPoliciesCtrl.ProcessNetworkPoliciesResponse(m.NetworkPoliciesResponse)
	default:
		return c.eventPipeline.Run(msg, c)
	}
}

func (c *sensorConnection) Run(server central.SensorService_CommunicateServer) error {
	if err := server.SendHeader(metadata.MD{}); err != nil {
		return errors.Wrap(err, "sending initial metadata")
	}

	go c.runSend(server)
	go c.handleMessages()
	c.runRecv(server)
	return c.stopSig.Err()
}

func (c *sensorConnection) ClusterID() string {
	return c.clusterID
}
