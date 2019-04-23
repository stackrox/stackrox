package deduper

import (
	"reflect"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

var (
	log = logging.LoggerForModule()
)

// key is the key by which messages are deduped.
type key struct {
	id           string
	resourceType reflect.Type
}

// deduper takes care of deduping sensor events.
type deduper struct {
	stream   messagestream.SensorMessageStream
	lastSent map[key]interface{}
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return deduper{
		stream:   stream,
		lastSent: make(map[key]interface{}),
	}
}

func (d deduper) Send(msg *central.MsgFromSensor) error {
	eventMsg, ok := msg.Msg.(*central.MsgFromSensor_Event)
	if !ok {
		// We only dedupe event messages, other messages get forwarded directly.
		return d.stream.Send(msg)
	}
	event := eventMsg.Event
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastSent, key)
		return d.stream.Send(msg)
	}

	if reflect.DeepEqual(d.lastSent[key], event) {
		return nil
	}

	if err := d.stream.Send(msg); err != nil {
		return err
	}
	// Make the action an update so we can dedupe CREATE and UPDATE
	event.Action = central.ResourceAction_UPDATE_RESOURCE
	d.lastSent[key] = event
	return nil
}
