package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
)

// GetAdministrationEvent returns a mock administration event.
func GetAdministrationEvent() *events.AdministrationEvent {
	return &events.AdministrationEvent{
		Domain:       "sample domain",
		Hint:         "sample hint",
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:      "sample message",
		ResourceID:   "some id",
		ResourceType: "Image",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
	}
}

// GetMultipleAdministrationEvents returns the given number of administration event.
// Each administration event will have a unique domain, hint, message, resource ID with its number as suffix.
// The resource type, level, and type are flipped between even / odd number of administration event:
// - Even ones will have resource type "Image", level "ERROR", and type "LOG_MESSAGE".
// - odd ones will have resource type "General", level "WARNING", and type "GENERIC".
func GetMultipleAdministrationEvents(numOfEvents int) []*events.AdministrationEvent {
	res := make([]*events.AdministrationEvent, 0, numOfEvents)
	for i := 0; i < numOfEvents; i++ {
		event := &events.AdministrationEvent{
			Domain:     fmt.Sprintf("sample domain %d", i),
			Hint:       fmt.Sprintf("sample hint %d", i),
			Message:    fmt.Sprintf("sample message %d", i),
			ResourceID: fmt.Sprintf("some resource ID %d", i),
		}
		if i%2 == 0 {
			event.ResourceType = "Image"
			event.Level = storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR
			event.Type = storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE
		} else {
			event.ResourceType = "General"
			event.Level = storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING
			event.Type = storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC
		}
		res = append(res, event)
	}
	return res
}

// GetListAdministrationEvents returns a set of events used for testing querying and searching.
// It consists of:
// - 3 Events with the domain "General", 3 Events with the domain "Image Scanning".
// - 3 Events with the resource "Image", 3 Events with the resource "Node".
// - 3 Events with the level "WARNING", 3 Events with the level "ERROR".
// - 3 Events with the type "LOG_MESSAGE", 3 Events with the type "GENERIC".
func GetListAdministrationEvents() []*events.AdministrationEvent {
	return []*events.AdministrationEvent{
		{
			Domain:       "General",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
			Message:      "sample message1",
			ResourceID:   "sample resource id",
			ResourceType: "Image",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		},
		{
			Domain:       "Image Scanning",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
			Message:      "sample message2",
			ResourceID:   "sample resource id",
			ResourceType: "Image",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		},
		{
			Domain:       "General",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
			Message:      "sample message3",
			ResourceID:   "sample resource id",
			ResourceType: "Node",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		},
		{
			Domain:       "Image Scanning",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
			Message:      "sample message4",
			ResourceID:   "sample resource id",
			ResourceType: "Image",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		},
		{
			Domain:       "General",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
			Message:      "sample message5",
			ResourceID:   "sample resource id",
			ResourceType: "Node",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		},
		{
			Domain:       "Image Scanning",
			Hint:         "sample hint",
			Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
			Message:      "sample message6",
			ResourceID:   "sample resource id",
			ResourceType: "Node",
			Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		},
	}
}
