package deduper

import (
	"fmt"
	"reflect"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/logging"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/managedcentral"
	"github.com/stackrox/rox/sensor/common/messagestream"
)

var (
	log = logging.LoggerForModule()
)

var (
	// TypeNetworkPolicy represents a NetworkPolicy Type
	TypeNetworkPolicy = reflect.TypeOf(&central.SensorEvent_NetworkPolicy{})
	// TypeDeployment represents a Deployment Type
	TypeDeployment = reflect.TypeOf(&central.SensorEvent_Deployment{})
	// TypePod represents a Pod Type
	TypePod = reflect.TypeOf(&central.SensorEvent_Pod{})
	// TypeNamespace represents a Namespace Type
	TypeNamespace = reflect.TypeOf(&central.SensorEvent_Namespace{})
	// TypeSecret represents a Secret Type
	TypeSecret = reflect.TypeOf(&central.SensorEvent_Secret{})
	// TypeNode represents a Node Type
	TypeNode = reflect.TypeOf(&central.SensorEvent_Node{})
	// TypeNodeInventory represents a NodeInventory Type
	TypeNodeInventory = reflect.TypeOf(&central.SensorEvent_NodeInventory{})
	// TypeServiceAccount represents a ServiceAccount Type
	TypeServiceAccount = reflect.TypeOf(&central.SensorEvent_ServiceAccount{})
	// TypeRole represents a Role Type
	TypeRole = reflect.TypeOf(&central.SensorEvent_Role{})
	// TypeBinding represents a Binding Type
	TypeBinding = reflect.TypeOf(&central.SensorEvent_Binding{})
	// TypeProcessIndicator represents a ProcessIndicator Type
	TypeProcessIndicator = reflect.TypeOf(&central.SensorEvent_ProcessIndicator{})
	// TypeProviderMetadata represents a ProviderMetadata Type
	TypeProviderMetadata = reflect.TypeOf(&central.SensorEvent_ProviderMetadata{})
	// TypeOrchestratorMetadata represents a OrchestratorMetadata Type
	TypeOrchestratorMetadata = reflect.TypeOf(&central.SensorEvent_OrchestratorMetadata{})
	// TypeImageIntegration represents a ImageIntegration Type
	TypeImageIntegration = reflect.TypeOf(&central.SensorEvent_ImageIntegration{})
	// TypeComplianceOperatorResult represents a ComplianceOperatorResult Type
	TypeComplianceOperatorResult = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorResult{})
	// TypeComplianceOperatorProfile represents a ComplianceOperatorProfile Type
	TypeComplianceOperatorProfile = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorProfile{})
	// TypeComplianceOperatorRule represents a ComplianceOperatorRule Type
	TypeComplianceOperatorRule = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorRule{})
	// TypeComplianceOperatorScanSettingBinding represents a ComplianceOperatorScanSettingBinding Type
	TypeComplianceOperatorScanSettingBinding = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorScanSettingBinding{})
	// TypeComplianceOperatorScan represents a ComplianceOperatorScan Type
	TypeComplianceOperatorScan = reflect.TypeOf(&central.SensorEvent_ComplianceOperatorScan{})
)

// deduper takes care of deduping sensor events.
type deduper struct {
	stream         messagestream.SensorMessageStream
	lastSent       map[deduperkey.Key]uint64
	centralState   set.StringSet
	observationSet *CloseableSet

	hasher *hash.Hasher
}

// NewDedupingMessageStream wraps a SensorMessageStream and dedupes events. Other message types are forwarded as-is.
func NewDedupingMessageStream(stream messagestream.SensorMessageStream, deduperState map[deduperkey.Key]uint64, observationSet *CloseableSet) messagestream.SensorMessageStream {
	if deduperState == nil {
		deduperState = make(map[deduperkey.Key]uint64)
	}

	centralOriginalState := set.NewStringSet()
	for k := range deduperState {
		centralOriginalState.Add(k.String())
	}

	return &deduper{
		stream:         stream,
		centralState:   centralOriginalState,
		lastSent:       deduperState,
		hasher:         hash.NewHasher(),
		observationSet: observationSet,
	}
}

func buildKey(typ, id string) string {
	return fmt.Sprintf("%s:%s", typ, id)
}

func getKey(msg *central.MsgFromSensor) string {
	event := msg.GetEvent()
	return buildKey(eventPkg.GetEventTypeWithoutPrefix(event.GetResource()), event.GetId())
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
	key := deduperkey.Key{
		ID:           event.GetId(),
		ResourceType: reflect.TypeOf(event.GetResource()),
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
			// If this is a SYNC event, we have to keep track of this event
			if msg.GetEvent().GetAction() == central.ResourceAction_SYNC_RESOURCE {
				key := getKey(msg)
				if d.centralState.Contains(key) {
					d.observationSet.AddIfOpen(getKey(msg))
				}
			}
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
