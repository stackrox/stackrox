package manager

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/central/sensorevent/service/streamer"
	"github.com/stackrox/rox/central/sensornetworkflow"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type sensorConnection struct {
	sendChan     chan *central.MsgToSensor
	eventsC      chan *v1.SensorEvent
	flowUpdatesC chan *central.NetworkFlowUpdate

	stopSig concurrency.ErrorSignal
}

func newConnection() *sensorConnection {
	conn := &sensorConnection{
		sendChan:     make(chan *central.MsgToSensor),
		eventsC:      make(chan *v1.SensorEvent),
		flowUpdatesC: make(chan *central.NetworkFlowUpdate),
		stopSig:      concurrency.NewErrorSignal(),
	}

	return conn
}

func (c *sensorConnection) runEventStreamer(eventStreamer streamer.Streamer) {
	eventStreamer.WaitUntilFinished()
	c.stopSig.SignalWithError(errors.New("event streamer terminated"))
}

func (c *sensorConnection) runFlowHandler(flowHandler sensornetworkflow.Handler) {
	err := flowHandler.Run()
	c.stopSig.SignalWithError(fmt.Errorf("network flow handler terminated with error: %v", err))
}

func (c *sensorConnection) waitForEventStreamer(eventStreamer streamer.Streamer) {
	eventStreamer.WaitUntilFinished()
	c.stopSig.SignalWithError(errors.New("event streamer terminated"))
}

func (c *sensorConnection) Communicate(server central.SensorService_CommunicateServer) error {
	err := c.doCommunicate(server)
	c.stopSig.SignalWithError(err)
	return err
}
func (c *sensorConnection) doCommunicate(server central.SensorService_CommunicateServer) error {
	recvChan := make(chan *central.MsgFromSensor)
	go c.handleRecv(server, recvChan)

	for {
		select {
		case msg := <-c.sendChan:
			if err := server.Send(msg); err != nil {
				return fmt.Errorf("send error: %v", err)
			}
		case msg := <-recvChan:
			if err := c.dispatchMsgFromSensor(msg); err != nil {
				return fmt.Errorf("dispatch error: %v", err)
			}
		case <-server.Context().Done():
			return fmt.Errorf("context error: %v", server.Context().Err())
		case <-c.stopSig.Done():
			return c.stopSig.Err()
		}
	}
}

func (c *sensorConnection) dispatchMsgFromSensor(msg *central.MsgFromSensor) error {
	switch m := msg.Msg.(type) {
	case *central.MsgFromSensor_Event:
		return c.injectEvent(m.Event)
	case *central.MsgFromSensor_NetworkFlowUpdate:
		return c.injectNetworkFlowUpdate(m.NetworkFlowUpdate)
	default:
		return fmt.Errorf("received unknown message type from sensor: %T", m)
	}
}

func (c *sensorConnection) handleRecv(server central.SensorService_CommunicateServer, recvChan chan *central.MsgFromSensor) {
	for !c.stopSig.IsDone() {
		msg, err := server.Recv()
		if err != nil {
			c.stopSig.SignalWithError(fmt.Errorf("receive failed: %v", err))
			return
		}

		select {
		case recvChan <- msg:
		case <-c.stopSig.Done():
			return
		}
	}
}
