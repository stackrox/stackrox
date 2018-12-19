package metrics

import (
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

func incrementSensorEvents(event *v1.SensorEvent, typ string) {
	resourceType := "none"
	if event.GetResource() != nil {
		resourceType = strings.TrimPrefix(reflect.TypeOf(event.GetResource()).Elem().Name(), "SensorEvent_")
	}
	labels := prometheus.Labels{
		"Action":       event.GetAction().String(),
		"ResourceType": resourceType,
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

func (s countingMessageStream) SendRaw(msg *central.MsgFromSensor, raw []byte) error {
	s.updateMetrics(msg)
	return s.stream.SendRaw(msg, raw)
}

// NewCountingEventStream returns a new SensorMessageStream that automatically updates metrics counters.
func NewCountingEventStream(stream messagestream.SensorMessageStream, typ string) messagestream.SensorMessageStream {
	return countingMessageStream{
		typ:    typ,
		stream: stream,
	}
}
