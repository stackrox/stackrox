package event

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/stringutils"
)

// GetEventTypeWithoutPrefix trims the *central.SensorEvent_ from the event type
func GetEventTypeWithoutPrefix(i interface{}) string {
	return stringutils.GetAfter(reflectutils.Type(i), "_")
}

// ParseIDFromKey returns the uuid portion of a key formatted with FormatKey.
func ParseIDFromKey(key string) string {
	return stringutils.GetAfter(key, ":")
}

// FormatKey formats a sensor event unique key formatted as <TYPE>:<UUID>
func FormatKey(typ, id string) string {
	return fmt.Sprintf("%s:%s", typ, id)
}

// GetKeyFromMessage generates an unique key from event resource type and event ID.
func GetKeyFromMessage(msg *central.MsgFromSensor) string {
	event := msg.GetEvent()
	return FormatKey(GetEventTypeWithoutPrefix(event.GetResource()), event.GetId())
}
