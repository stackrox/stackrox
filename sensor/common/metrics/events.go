package metrics

import (
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/sensor/common/eventstream"
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

type countingEventStream struct {
	typ    string
	stream eventstream.SensorEventStream
}

func (s countingEventStream) Send(event *v1.SensorEvent) error {
	incrementSensorEvents(event, s.typ)
	return s.stream.Send(event)
}

func (s countingEventStream) SendRaw(event *v1.SensorEvent, raw []byte) error {
	incrementSensorEvents(event, s.typ)
	return s.stream.SendRaw(event, raw)
}

// NewCountingEventStream returns a new SensorEventStream that automatically updates metrics counters.
func NewCountingEventStream(stream eventstream.SensorEventStream, typ string) eventstream.SensorEventStream {
	return countingEventStream{
		typ:    typ,
		stream: stream,
	}
}
