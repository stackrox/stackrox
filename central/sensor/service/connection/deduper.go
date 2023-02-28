package connection

import (
	"reflect"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/sensor/hash"
)

type key struct {
	id           string
	resourceType reflect.Type
}

func newDeduper() *deduper {
	return &deduper{
		lastReceived: make(map[key]uint64),
		hasher:       hash.NewHasher(),
	}
}

type deduper struct {
	lastReceived map[key]uint64

	hasher *hash.Hasher
}

func skipDedupe(msg *central.MsgFromSensor) bool {
	eventMsg, ok := msg.Msg.(*central.MsgFromSensor_Event)
	if !ok {
		return true
	}
	if eventMsg.Event.GetProcessIndicator() != nil {
		return true
	}
	if alert.IsRuntimeAlertResult(msg.GetEvent().GetAlertResults()) {
		return true
	}
	if eventMsg.Event.GetReprocessDeployment() != nil {
		return true
	}
	return false
}

func (d *deduper) dedupe(msg *central.MsgFromSensor) bool {
	if skipDedupe(msg) {
		return false
	}
	event := msg.GetEvent()
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastReceived, key)
		return false
	}
	receivedHash := event.GetSensorHash()
	// Backwards compatibility with a previous Sensor
	if event.GetSensorHashOneof() == nil {
		// Compute the sensor hash
		hashValue, ok := d.hasher.HashMsg(msg)
		if !ok {
			return false
		}
		receivedHash = hashValue
	}
	prevValue, ok := d.lastReceived[key]
	if ok && prevValue == receivedHash {
		return true
	}
	d.lastReceived[key] = receivedHash
	return false
}
