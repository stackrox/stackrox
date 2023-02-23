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

func (d *deduper) shouldReprocess(msg *central.MsgFromSensor) bool {
	event := msg.GetEvent()
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		return true
	}
	prevValue, ok := d.lastReceived[key]
	if !ok {
		// This implies that a REMOVE event has been processed before this event
		// Note: we may want to handle alerts specifically because we should insert them as already resolved for completeness
		return false
	}
	// This implies that no new event was processed after the initial processing of the current message
	return prevValue == event.GetSensorHash()
}

func (d *deduper) shouldProcess(msg *central.MsgFromSensor) bool {
	if skipDedupe(msg) {
		return true
	}
	if msg.GetProcessingAttempt() > 0 {
		return d.shouldReprocess(msg)
	}

	event := msg.GetEvent()
	key := key{
		id:           event.GetId(),
		resourceType: reflect.TypeOf(event.GetResource()),
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		delete(d.lastReceived, key)
		return true
	}
	// Backwards compatibility with a previous Sensor
	if event.GetSensorHashOneof() == nil {
		// Compute the sensor hash
		hashValue, ok := d.hasher.HashEvent(msg.GetEvent())
		if !ok {
			return false
		}
		event.SensorHashOneof = &central.SensorEvent_SensorHash{
			SensorHash: hashValue,
		}
	}
	prevValue, ok := d.lastReceived[key]
	if ok && prevValue == event.GetSensorHash() {
		return false
	}
	d.lastReceived[key] = event.GetSensorHash()
	return true
}
