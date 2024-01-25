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
	key := s.metricKey(typ, eventType)
	s.maxSeen[key] = math.Max(s.maxSeen[key], float64(msg.Size()))
	sensorGRPCMaxMessageSize.With(prometheus.Labels{
		"Type":      typ,
		"EventType": eventType,
	}).Set(s.maxSeen[key])
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
	return &sizingEventStream{stream, map[string]float64{}}
}
