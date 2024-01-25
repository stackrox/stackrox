package metrics

import (
	"math"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

type sizingEventStream struct {
	stream  messagestream.SensorMessageStream
	maxSeen map[string]float64
}

func (s *sizingEventStream) incrementMetric(msg *central.MsgFromSensor) {
	typ := reflectutils.Type(msg)
	var eventType string
	if msg.GetEvent() != nil {
		eventType = event.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource())
	}
	gaugeValue := s.setIfHigher(s.metricKey(typ, eventType), float64(msg.Size()))
	sensorMessageSize.With(prometheus.Labels{
		"Type":      typ,
		"EventType": eventType,
	}).Set(gaugeValue)
}

// setIfHigher updates the map if size is higher than the maxSeen[key] value. Returns whichever is higher
func (s *sizingEventStream) setIfHigher(key string, size float64) float64 {
	if v, ok := s.maxSeen[key]; ok {
		size = math.Max(v, size)
	}
	s.maxSeen[key] = size
	return size
}

func (s *sizingEventStream) metricKey(typ, eventType string) string {
	return typ + "_" + eventType
}

func (s *sizingEventStream) Send(msg *central.MsgFromSensor) error {
	s.incrementMetric(msg)
	return s.stream.Send(msg)
}

// NewSizingEventStream returns a new SensorMessageStream that automatically updates size metrics.
func NewSizingEventStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return &sizingEventStream{stream: stream}
}
