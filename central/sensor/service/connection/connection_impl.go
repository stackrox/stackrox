package connection

import (
	"fmt"

	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type sensorConnection struct {
	clusterID           string
	stopSig, stoppedSig concurrency.ErrorSignal

	sendC chan *central.MsgToSensor

	scrapeCtrl scrape.Controller

	eventQueue    *dedupingQueue
	eventPipeline pipeline.ClusterPipeline
}

func newConnection(clusterID string, pf pipeline.Factory) (*sensorConnection, error) {
	eventPipeline, err := pf.PipelineForCluster(clusterID)
	if err != nil {
		return nil, fmt.Errorf("creating event pipeline: %v", err)
	}

	conn := &sensorConnection{
		stopSig:       concurrency.NewErrorSignal(),
		stoppedSig:    concurrency.NewErrorSignal(),
		sendC:         make(chan *central.MsgToSensor),
		eventPipeline: eventPipeline,
		eventQueue:    newDedupingQueue(),
	}

	conn.scrapeCtrl = scrape.NewController(conn, &conn.stopSig)
	return conn, nil
}

func (c *sensorConnection) Terminate(err error) bool {
	return c.stopSig.SignalWithError(err)
}

func (c *sensorConnection) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedSig
}

func (c *sensorConnection) runRecv(server central.SensorService_CommunicateServer) {
	for !c.stopSig.IsDone() {
		msg, err := server.Recv()
		if err != nil {
			c.stopSig.SignalWithError(fmt.Errorf("recv error: %v", err))
			return
		}

		c.eventQueue.push(msg)
	}
}

func (c *sensorConnection) handleMessages() {
	for msg := c.eventQueue.pullBlocking(&c.stopSig); msg != nil; msg = c.eventQueue.pullBlocking(&c.stopSig) {
		if err := c.handleMessage(msg); err != nil {
			log.Errorf("Error handling sensor message: %v", err)
		}
	}
	c.stoppedSig.SignalWithError(c.stopSig.Err())
}

func (c *sensorConnection) runSend(server central.SensorService_CommunicateServer) {
	for !c.stopSig.IsDone() {
		select {
		case <-c.stopSig.Done():
			return
		case <-server.Context().Done():
			c.stopSig.SignalWithError(fmt.Errorf("context error: %v", server.Context().Err()))
			return
		case msg := <-c.sendC:
			if err := server.Send(msg); err != nil {
				c.stopSig.SignalWithError(fmt.Errorf("send error: %v", err))
				return
			}
		}
	}
}

func (c *sensorConnection) Scrapes() scrape.Controller {
	return c.scrapeCtrl
}

func (c *sensorConnection) InjectMessage(msg *central.MsgToSensor) error {
	select {
	case c.sendC <- msg:
		return nil
	case <-c.stopSig.Done():
		return fmt.Errorf("could not send message as sensor connection was stopped: %v", c.stopSig.Err())
	}
}

func (c *sensorConnection) handleMessage(msg *central.MsgFromSensor) error {
	switch m := msg.Msg.(type) {
	case *central.MsgFromSensor_ScrapeUpdate:
		return c.scrapeCtrl.ProcessScrapeUpdate(m.ScrapeUpdate)
	default:
		return c.eventPipeline.Run(msg, c)
	}
}

func (c *sensorConnection) Run(server central.SensorService_CommunicateServer) error {
	go c.runSend(server)
	go c.handleMessages()
	c.runRecv(server)
	c.eventPipeline.OnFinish()
	return c.stopSig.Err()
}
