package metrics

import (
	"github.com/pkg/errors"
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
	if err := s.stream.Send(msg); err != nil {
		return errors.Wrap(err, "sending sensor message in timingMessageStream")
	}
	return nil
}

// NewTimingEventStream returns a new SensorMessageStream that automatically updates timing metrics.
func NewTimingEventStream(stream messagestream.SensorMessageStream, typ string) messagestream.SensorMessageStream {
	return timingMessageStream{
		stream: stream,
		typ:    typ,
	}
}
