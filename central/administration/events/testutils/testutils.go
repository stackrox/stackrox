package testutils

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
)

var (
	eventCounter = 0
)

// GenerateAdministrativeEvent creates an administrative event for testing.
func GenerateAdministrativeEvent(eventLevel storage.AdministrationEventLevel, domain string) *events.AdministrationEvent {
	event := &events.AdministrationEvent{
		Domain:       domain,
		Hint:         fmt.Sprintf("sample hint %d", eventCounter),
		Level:        eventLevel,
		Message:      fmt.Sprintf("sample message %d", eventCounter),
		ResourceID:   fmt.Sprintf("some resource ID %d", eventCounter),
		ResourceType: "Image",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
	}
	eventCounter++
	return event
}
