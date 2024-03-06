package events

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var rootNamespaceUUID = uuid.FromStringOrPanic("d4dcc3d8-fcdf-4621-8386-0be1372ecbba")

// GenerateEventID returns a deduplication ID as UUID5 based on the event content.
func GenerateEventID(event *AdministrationEvent) string {
	dedupKey := strings.Join([]string{
		event.GetDomain(),
		event.GetMessage(),
		stringutils.FirstNonEmpty(event.GetResourceID(), event.GetResourceName()),
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
	ResourceName string
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

// GetResourceName returns the event resource name.
func (m *AdministrationEvent) GetResourceName() string {
	if m != nil {
		return m.ResourceName
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
	tsNow := protocompat.TimestampNow()
	return &storage.AdministrationEvent{
		Id:      GenerateEventID(m),
		Type:    m.GetType(),
		Level:   m.GetLevel(),
		Message: m.GetMessage(),
		Hint:    m.GetHint(),
		Domain:  m.GetDomain(),
		Resource: &storage.AdministrationEvent_Resource{
			Type: m.GetResourceType(),
			Id:   m.GetResourceID(),
			Name: m.GetResourceName(),
		},
		NumOccurrences: 1,
		CreatedAt:      tsNow,
		LastOccurredAt: tsNow,
	}
}

// Validate will validate the administration event.
// Note that Validate may be called on a nil administration event.
func (m *AdministrationEvent) Validate() error {
	if m == nil {
		return errox.InvalidArgs.CausedBy("empty event given")
	}

	// This needs to be kept in-line with the fields used for generating the event ID (see GenerateEventID).
	if stringutils.AtLeastOneEmpty(m.GetDomain(),
		m.GetMessage(),
		stringutils.FirstNonEmpty(m.GetResourceID(), m.GetResourceName()),
		m.GetResourceType(),
		m.GetType().String()) {
		return errox.InvalidArgs.CausedBy("all required fields must be set")
	}

	return nil
}
