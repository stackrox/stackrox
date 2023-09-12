package events

import "github.com/stackrox/rox/generated/storage"

type AdministrationEvent struct {
	Domain       string
	Hint         string
	Level        storage.AdministrationEventLevel
	Message      string
	ResourceId   string
	ResourceType string
	Type         storage.AdministrationEventType
}

func (m *AdministrationEvent) GetDomain() string {
	if m != nil {
		return m.Domain
	}
	return ""
}

func (m *AdministrationEvent) GetHint() string {
	if m != nil {
		return m.Hint
	}
	return ""
}

func (m *AdministrationEvent) GetLevel() storage.AdministrationEventLevel {
	if m != nil {
		return m.Level
	}
	return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_UNKNOWN
}

func (m *AdministrationEvent) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *AdministrationEvent) GetResourceId() string {
	if m != nil {
		return m.ResourceId
	}
	return ""
}

func (m *AdministrationEvent) GetResourceType() string {
	if m != nil {
		return m.ResourceType
	}
	return ""
}

func (m *AdministrationEvent) GetType() storage.AdministrationEventType {
	if m != nil {
		return m.Type
	}
	return storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_UNKNOWN
}
