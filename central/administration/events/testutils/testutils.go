package testutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/uuid"
)

// GenerateAdministrativeEvent creates an administrative event for testing.
func GenerateAdministrativeEvent(eventLevel storage.AdministrationEventLevel) *events.AdministrationEvent {
	return &events.AdministrationEvent{
		ResourceID: uuid.NewV4().String(),
		Level:      eventLevel,
	}
}
