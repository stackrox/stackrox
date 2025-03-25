package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

// deduper takes care of deduping sensor events.
type streamDebugger struct {
	stream messagestream.SensorMessageStream
}

// NewDebuggingMessageStream wraps a SensorMessageStream and prints all messages to log
func NewDebuggingMessageStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return &streamDebugger{
		stream: stream,
	}
}

func (d *streamDebugger) Send(msg *central.MsgFromSensor) error {
	ty := stringutils.GetAfter(reflectutils.Type(msg.Msg), "_")
	log.Infof("TYPE=%s, MSG=%s\n", ty, msg.String())
	if err := d.stream.Send(msg); err != nil {
		return err
	}

	return nil
}
