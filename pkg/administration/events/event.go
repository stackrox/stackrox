package events

import (
	"strings"

	gogoTimestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

var rootNamespaceUUID = uuid.FromStringOrNil("d4dcc3d8-fcdf-4621-8386-0be1372ecbba")

// GenerateEventID returns a deduplication ID as UUID5 based on the event content.
func GenerateEventID(event *AdministrationEvent) string {
	dedupKey := strings.Join([]string{
		event.GetDomain(),
		event.GetMessage(),
		event.GetResourceID(),
		event.GetResourceType(),
		event.GetType().String(),
	}, ",")
	return uuid.NewV5(rootNamespaceUUID, dedupKey).String()
}

// AdministrationEvent contains a sub set of *storage.AdministrationEvent.
//
// Fields managed by the event service, such as the dedup ID and timestamps,
// are excluded.
type AdministrationEvent struct {
	Domain       string
	Hint         string
	Level        storage.AdministrationEventLevel
	Message      string
	ResourceID   string
	ResourceType string
	Type         storage.AdministrationEventType
}

// GetDomain returns the event domain.
func (m *AdministrationEvent) GetDomain() string {
	if m != nil {
		return m.Domain
	}
	return ""
}

// GetHint returns the event hint.
func (m *AdministrationEvent) GetHint() string {
	if m != nil {
		return m.Hint
	}
	return ""
}

// GetLevel returns the event level.
func (m *AdministrationEvent) GetLevel() storage.AdministrationEventLevel {
	if m != nil {
		return m.Level
	}
	return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_UNKNOWN
}

// GetMessage returns the event message.
func (m *AdministrationEvent) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

// GetResourceID returns the event resource ID.
func (m *AdministrationEvent) GetResourceID() string {
	if m != nil {
		return m.ResourceID
	}
	return ""
}

// GetResourceType returns the event resource type.
func (m *AdministrationEvent) GetResourceType() string {
	if m != nil {
		return m.ResourceType
	}
	return ""
}

// GetType returns the event type.
func (m *AdministrationEvent) GetType() storage.AdministrationEventType {
	if m != nil {
		return m.Type
	}
	return storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_UNKNOWN
}

// ToStorageEvent converts the event to its storage representation.
func (m *AdministrationEvent) ToStorageEvent() *storage.AdministrationEvent {
	tsNow := gogoTimestamp.TimestampNow()
	return &storage.AdministrationEvent{
		Id:             GenerateEventID(m),
		Type:           m.GetType(),
		Level:          m.GetLevel(),
		Message:        m.GetMessage(),
		Hint:           m.GetHint(),
		Domain:         m.GetDomain(),
		ResourceId:     m.GetResourceID(),
		ResourceType:   m.GetResourceType(),
		NumOccurrences: 1,
		CreatedAt:      tsNow,
		LastOccurredAt: tsNow,
	}
}
