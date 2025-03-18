package testutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/fixtures"
)

// GenerateAdministrativeEvent creates an administrative event for testing.
func GenerateAdministrativeEvent(eventLevel storage.AdministrationEventLevel) *events.AdministrationEvent {
	event := fixtures.GetAdministrationEvent()
	event.Level = eventLevel
	return event
}
