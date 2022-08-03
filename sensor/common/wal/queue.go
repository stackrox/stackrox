package wal

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

var (
	log = logging.LoggerForModule()
)

// deduper takes care of deduping sensor events.
type deduper struct {
	stream       messagestream.SensorMessageStream
	messageAcker MessageAcker
}

// NewDataStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDataStream(stream messagestream.SensorMessageStream, msgAcker MessageAcker) messagestream.SensorMessageStream {
	return &deduper{
		stream:       stream,
		messageAcker: msgAcker,
	}
}

func (d *deduper) Send(msg *central.MsgFromSensor) error {
	event := msg.GetEvent()
	if event.GetHasHash() == nil && event.GetAction() != central.ResourceAction_REMOVE_RESOURCE {
		return d.stream.Send(msg)
	}
	d.messageAcker.Insert(event)
	return d.stream.Send(msg)
}
