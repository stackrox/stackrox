package deduper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
)

var (
	deduperTypes = []any{
		&central.SensorEvent_NetworkPolicy{},
		&central.SensorEvent_Deployment{},
		&central.SensorEvent_Pod{},
		&central.SensorEvent_Namespace{},
		&central.SensorEvent_Secret{},
		&central.SensorEvent_Node{},
		&central.SensorEvent_ServiceAccount{},
		&central.SensorEvent_Role{},
		&central.SensorEvent_Binding{},
		&central.SensorEvent_NodeInventory{},
		&central.SensorEvent_ProcessIndicator{},
		&central.SensorEvent_ProviderMetadata{},
		&central.SensorEvent_OrchestratorMetadata{},
		&central.SensorEvent_ImageIntegration{},
		&central.SensorEvent_ComplianceOperatorResult{},
		&central.SensorEvent_ComplianceOperatorProfile{},
		&central.SensorEvent_ComplianceOperatorRule{},
		&central.SensorEvent_ComplianceOperatorScanSettingBinding{},
		&central.SensorEvent_ComplianceOperatorScan{},
		&central.SensorEvent_AlertResults{},
	}
)

// Key by which messages are deduped.
type Key struct {
	ID           string
	ResourceType reflect.Type
}

// ParseDeduperState makes a copy of the deduper state.
func ParseDeduperState(state map[string]uint64) map[Key]uint64 {
	if state == nil {
		return make(map[Key]uint64)
	}

	result := make(map[Key]uint64, len(state))
	for k, v := range state {
		parsedKey, err := keyFrom(k)
		if err != nil {
			log.Warnf("Deduper state has malformed entry: %s->%d: %s", k, v, err)
			continue
		}
		result[parsedKey] = v
	}
	return result
}

func keyFrom(v string) (Key, error) {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return Key{}, fmt.Errorf("invalid Key format: %s", v)
	}
	t, err := mapType(parts[0])
	if err != nil {
		return Key{}, errors.Wrap(err, "map type")
	}
	return Key{
		ID:           parts[1],
		ResourceType: t,
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
