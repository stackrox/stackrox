package events

import "github.com/stackrox/rox/generated/storage"

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
