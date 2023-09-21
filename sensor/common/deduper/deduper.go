package deduper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/logging"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/sensor/common/managedcentral"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

var (
	log = logging.LoggerForModule()

	deduperTypes = []any{
		storage.NetworkPolicy{},
		storage.Deployment{},
		storage.Pod{},
		storage.NamespaceMetadata{},
		storage.Secret{},
		storage.Node{},
		storage.NodeInventory{},
		storage.ServiceAccount{},
		storage.K8SRole{},
		storage.K8SRoleBinding{},
		storage.ProcessIndicator{},
		storage.ProviderMetadata{},
		storage.OrchestratorMetadata{},
		storage.ImageIntegration{},
		storage.ComplianceOperatorCheckResult{},
		storage.ComplianceOperatorProfile{},
		storage.ComplianceOperatorRule{},
		storage.ComplianceOperatorScanSettingBinding{},
		storage.ComplianceOperatorScan{},
	}
)

// key is the key by which messages are deduped.
type key struct {
	id           string
	resourceType reflect.Type
}

func keyFrom(v string) (key, error) {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return key{}, fmt.Errorf("invalid key format: %s", v)
	}
	t, err := mapType(parts[0])
	if err != nil {
		return key{}, errors.Wrap(err, "map type")
	}
	return key{
		id:           parts[1],
		resourceType: t,
	}, nil
}

func mapType(typeStr string) (reflect.Type, error) {
	for _, t := range deduperTypes {
		if typeStr == eventPkg.GetEventTypeWithoutPrefix(t) {
			return reflect.TypeOf(t), nil
		}
	}
	return nil, fmt.Errorf("invalid type: %s", typeStr)
}

// deduper takes care of deduping sensor events.
type deduper struct {
	stream   messagestream.SensorMessageStream
	lastSent map[key]uint64

	hasher *hash.Hasher
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream, deduperState map[string]uint64) messagestream.SensorMessageStream {
	lastSeen := copyDeduperState(deduperState)

	return &deduper{
		stream:   stream,
		lastSent: lastSeen,
		hasher:   hash.NewHasher(),
	}
}

func copyDeduperState(state map[string]uint64) map[key]uint64 {
	if state == nil {
		return make(map[key]uint64)
	}

	result := make(map[key]uint64, len(state))
	for k, v := range state {
		parsedKey, err := keyFrom(k)
		if err != nil {
			log.Warnf("Deduper state received from central has malformed entry: %s->%d: %s", k, v, err)
			continue
		}
		result[parsedKey] = v
	}
	return result
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
