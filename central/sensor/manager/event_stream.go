package manager

import (
	"io"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
)

type eventStream struct {
	conn *sensorConnection
}

func (c *sensorConnection) newEventStream() *eventStream {
	return &eventStream{
		conn: c,
	}
}

func (s *eventStream) Send(enforcement *v1.SensorEnforcement) error {
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_Enforcement{
			Enforcement: enforcement,
		},
	}

	if err := s.conn.stopSig.ErrorWithDefault(io.EOF); err != nil {
		return err
	}

	select {
	case s.conn.sendChan <- msg:
		return nil
	case <-s.conn.stopSig.Done():
		return s.conn.stopSig.Err()
	}
}

func (c *sensorConnection) injectEvent(event *v1.SensorEvent) error {
	if err := c.stopSig.ErrorWithDefault(io.EOF); err != nil {
		return err
	}

	select {
	case c.eventsC <- event:
		return nil
	case <-c.stopSig.Done():
		return c.stopSig.Err()
	}
}

func (s *eventStream) Recv() (*v1.SensorEvent, error) {
	if err := s.conn.stopSig.ErrorWithDefault(io.EOF); err != nil {
		return nil, err
	}

	select {
	case event := <-s.conn.eventsC:
		return event, nil
	case <-s.conn.stopSig.Done():
		return nil, s.conn.stopSig.Err()
	}
}
