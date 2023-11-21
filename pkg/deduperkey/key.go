package deduperkey

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log          = logging.LoggerForModule()
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

// String returns the string version of the key
func (k *Key) String() string {
	typ := stringutils.GetAfter(k.ResourceType.String(), "_")
	return eventPkg.FormatKey(typ, k.ID)
}

// ParseKeySlice returns a list of Key objects from a list a string formatted keys. An error returned means that some
// of the keys might have failed when being parsed.
func ParseKeySlice(keys []string) ([]Key, error) {
	errList := errorhelpers.NewErrorList("malformed key entries")
	result := make([]Key, len(keys))
	for i, v := range keys {
		parsedKey, err := KeyFrom(v)
		if err != nil {
			errList.AddError(errors.Wrapf(err, "key: %s", v))
			continue
		}
		result[i] = parsedKey
	}
	return result, errList.ToError()
}

// ParseDeduperState makes a copy of the deduper state.
func ParseDeduperState(state map[string]uint64) map[Key]uint64 {
	if state == nil {
		return make(map[Key]uint64)
	}

	result := make(map[Key]uint64, len(state))
	for k, v := range state {
		parsedKey, err := KeyFrom(k)
		if err != nil {
			log.Warnf("Deduper state has malformed entry: %s->%d: %s", k, v, err)
			continue
		}
		result[parsedKey] = v
	}
	return result
}

// KeyFrom parses a string key into Key
func KeyFrom(v string) (Key, error) {
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
