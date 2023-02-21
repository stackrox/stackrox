package connection

import (
	"hash"
	"hash/fnv"
	"reflect"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/utils"
)

type key struct {
	id           string
	resourceType reflect.Type
}

func newDeduper() *deduper {
	return &deduper{
		lastReceived: make(map[key]uint64),
		hasher:       fnv.New64a(),
	}
}

type deduper struct {
	lastReceived map[key]uint64

	hasher hash.Hash64
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
		d.hasher.Reset()
		hashValue, err := hashstructure.Hash(event.GetResource(), hashstructure.FormatV2, &hashstructure.HashOptions{
			TagName: "sensorhash",
			Hasher:  d.hasher,
		})
		utils.Should(err)

		receivedHash = hashValue
	}
	prevValue, ok := d.lastReceived[key]
	if ok && prevValue == receivedHash {
		return true
	}
	d.lastReceived[key] = receivedHash
	return false
}
