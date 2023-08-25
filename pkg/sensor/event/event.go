package event

import (
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/stringutils"
)

// GetEventTypeWithoutPrefix trims the *central.SensorEvent_ from the event type
func GetEventTypeWithoutPrefix(i interface{}) string {
	return stringutils.GetAfter(reflectutils.Type(i), "_")
}
