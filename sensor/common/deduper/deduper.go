package deduper

import (
	"reflect"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/sensor/common/managedcentral"
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
	lastSent map[key]uint64

	hasher *hash.Hasher
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream, deduperState *central.DeduperState) messagestream.SensorMessageStream {

	return &deduper{
		stream:   stream,
		lastSent: make(map[key]uint64),
		hasher:   hash.NewHasher(),
	}
}

func (d *deduper) Send(msg *central.MsgFromSensor) error {
	eventMsg, ok := msg.Msg.(*central.MsgFromSensor_Event)
	if !ok || eventMsg.Event.GetProcessIndicator() != nil || alert.IsRuntimeAlertResult(msg.GetEvent().GetAlertResults()) {
		// We only dedupe event messages (excluding process indicators and runtime alerts which are always unique), other messages get forwarded directly.
		return d.stream.Send(msg)
	}
	event := eventMsg.Event
	// This filter works around race conditions in which image integrations may be initialized prior to CentralHello being received
	if managedcentral.IsCentralManaged() && event.GetImageIntegration() != nil {
		return nil
	}
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		priorLen := len(d.lastSent)
		delete(d.lastSent, key)
		// Do not send a remove message for something that has not been seen before
		// This also effectively dedupes REMOVE actions
		if priorLen == len(d.lastSent) {
			return nil
		}
		return d.stream.Send(msg)
	}

	hashValue, ok := d.hasher.HashEvent(msg.GetEvent())
	if ok {
		// If the hash is valid, then check for deduping
		if d.lastSent[key] == hashValue {
			return nil
		}
		event.SensorHashOneof = &central.SensorEvent_SensorHash{
			SensorHash: hashValue,
		}
		d.lastSent[key] = hashValue
	}

	if err := d.stream.Send(msg); err != nil {
		return err
	}

	return nil
}
