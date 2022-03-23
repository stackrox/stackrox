package deduper

import (
	"hash"
	"hash/fnv"
	"reflect"

	"github.com/mitchellh/hashstructure"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
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
	hasher   hash.Hash64
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream) messagestream.SensorMessageStream {
	return &deduper{
		stream:   stream,
		lastSent: make(map[key]uint64),
		hasher:   fnv.New64a(),
	}
}

func isRuntimeAlert(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetAlertResults().GetStage() == storage.LifecycleStage_RUNTIME
}

func (d *deduper) Send(msg *central.MsgFromSensor) error {
	eventMsg, ok := msg.Msg.(*central.MsgFromSensor_Event)
	if !ok || eventMsg.Event.GetProcessIndicator() != nil || isRuntimeAlert(msg) {
		// We only dedupe event messages (excluding process indicators and runtime alerts which are always unique), other messages get forwarded directly.
		return d.stream.Send(msg)
	}
	event := eventMsg.Event
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

	d.hasher.Reset()
	hashValue, err := hashstructure.Hash(event.GetResource(), &hashstructure.HashOptions{
		TagName: "sensorhash",
		Hasher:  d.hasher,
	})
	utils.Should(err)

	if d.lastSent[key] == hashValue {
		return nil
	}

	if err := d.stream.Send(msg); err != nil {
		return err
	}
	d.lastSent[key] = hashValue

	return nil
}
