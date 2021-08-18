package metrics

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

type timingMessageStream struct {
	stream messagestream.SensorMessageStream
	typ    string
}

func (s timingMessageStream) Send(msg *central.MsgFromSensor) error {
	metrics.SetResourceProcessingDurationForEvent(k8sObjectIngestionToSendDuration, msg.GetEvent(), s.typ)
	return s.stream.Send(msg)
}

// NewTimingEventStream returns a new SensorMessageStream that automatically updates timing metrics.
func NewTimingEventStream(stream messagestream.SensorMessageStream, typ string) messagestream.SensorMessageStream {
	return timingMessageStream{
		stream: stream,
		typ:    typ,
	}
}
