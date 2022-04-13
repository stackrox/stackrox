package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/sensor/common/messagestream"
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
	return s.stream.Send(msg)
}

// NewCountingEventStream returns a new SensorMessageStream that automatically updates metrics counters.
func NewCountingEventStream(stream messagestream.SensorMessageStream, typ string) messagestream.SensorMessageStream {
	return countingMessageStream{
		typ:    typ,
		stream: stream,
	}
}
