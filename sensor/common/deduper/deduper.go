package deduper

import (
	"bytes"
	"reflect"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/eventstream"
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
	stream   eventstream.SensorEventStream
	lastSent map[key][]byte
}

// NewDedupingEventStream wraps
func NewDedupingEventStream(stream eventstream.SensorEventStream) eventstream.SensorEventStream {
	return deduper{
		stream:   stream,
		lastSent: make(map[key][]byte),
	}
}

func (d deduper) Send(event *v1.SensorEvent) error {
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastSent, key)
	}
	if event.GetAction() != v1.ResourceAction_UPDATE_RESOURCE {
		return d.stream.Send(event)
	}

	serialized, err := serializeDeterministic(event)
	if err != nil {
		log.Warnf("Could not deterministically serialize event: %v", err)
		delete(d.lastSent, key)
		return d.stream.Send(event)
	}

	return d.doSendRaw(event, key, serialized)
}

func (d deduper) SendRaw(event *v1.SensorEvent, raw []byte) error {
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastSent, key)
	}
	if event.GetAction() != v1.ResourceAction_UPDATE_RESOURCE {
		return d.stream.SendRaw(event, raw)
	}

	return d.doSendRaw(event, key, raw)
}

func (d deduper) doSendRaw(event *v1.SensorEvent, key key, serialized []byte) error {
	if bytes.Equal(d.lastSent[key], serialized) {
		return nil
	}
	d.lastSent[key] = serialized

	return d.stream.SendRaw(event, serialized)
}
