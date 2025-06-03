package metrics

import (
	"math"

	"github.com/pkg/errors"
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
	messageType := reflectutils.Type(msg.GetMsg())
	var eventType string
	if msg.GetEvent() != nil {
		eventType = event.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource())
	}
	messageType = s.metricKey(messageType, eventType)

	messageSize := float64(msg.SizeVT())
	labels := prometheus.Labels{
		"Type": messageType,
	}

	sensorMessageSizeSent.With(labels).Observe(messageSize)
	sensorLastMessageSizeSent.With(labels).Set(messageSize)

	s.maxSeen[messageType] = math.Max(s.maxSeen[messageType], messageSize)
	sensorMaxMessageSizeSent.With(labels).Set(s.maxSeen[messageType])
}

func (s *sizingEventStream) metricKey(typ, eventType string) string {
	return typ + "_" + eventType
}

func (s *sizingEventStream) Send(msg *central.MsgFromSensor) error {
	s.incrementMetric(msg)
	if err := s.stream.Send(msg); err != nil {
		return errors.Wrap(err, "sending sensor message in sizingEventStream")
	}
	return nil
}

// NewSizingEventStream returns a new SensorMessageStream that automatically updates max message size sent metric.
func NewSizingEventStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return &sizingEventStream{stream, make(map[string]float64)}
}
