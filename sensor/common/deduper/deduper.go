package deduper

import (
	"bytes"
	"reflect"

	"github.com/stackrox/rox/generated/api/v1"
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
	lastSent map[key][]byte
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return deduper{
		stream:   stream,
		lastSent: make(map[key][]byte),
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
	if event.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastSent, key)
	}
	if event.GetAction() != v1.ResourceAction_UPDATE_RESOURCE {
		return d.stream.Send(msg)
	}

	serialized, err := serializeDeterministic(msg)
	if err != nil {
		log.Warnf("Could not deterministically serialize event: %v", err)
		delete(d.lastSent, key)
		return d.stream.Send(msg)
	}

	return d.doSendRaw(msg, key, serialized)
}

func (d deduper) SendRaw(msg *central.MsgFromSensor, raw []byte) error {
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
	if event.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastSent, key)
	}
	if event.GetAction() != v1.ResourceAction_UPDATE_RESOURCE {
		return d.stream.SendRaw(msg, raw)
	}

	return d.doSendRaw(msg, key, raw)
}

func (d deduper) doSendRaw(msg *central.MsgFromSensor, key key, serialized []byte) error {
	if bytes.Equal(d.lastSent[key], serialized) {
		return nil
	}
	d.lastSent[key] = serialized

	return d.stream.SendRaw(msg, serialized)
}
