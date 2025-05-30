package metrics

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

func incrementSensorEvents(event *central.SensorEvent, typ string) {
	labels := prometheus.Labels{
		"Action":       event.GetAction().String(),
		"ResourceType": metrics.GetResourceString(event),
		"Type":         typ,
	}
	sensorEvents.With(labels).Inc()
}

type countingMessageStream struct {
	typ    string
	stream messagestream.SensorMessageStream
}

func (s countingMessageStream) updateMetrics(msg *central.MsgFromSensor) {
	switch m := msg.Msg.(type) {
	case *central.MsgFromSensor_Event:
		incrementSensorEvents(m.Event, s.typ)
	default:
		// we take care of metrics for network flows elsewhere
	}
}

func (s countingMessageStream) Send(msg *central.MsgFromSensor) error {
	s.updateMetrics(msg)
	if err := s.stream.Send(msg); err != nil {
		return errors.Wrap(err, "sending sensor message")
	}
	return nil
}

// NewCountingEventStream returns a new SensorMessageStream that automatically updates metrics counters.
func NewCountingEventStream(stream messagestream.SensorMessageStream, typ string) messagestream.SensorMessageStream {
	return countingMessageStream{
		typ:    typ,
		stream: stream,
	}
}
