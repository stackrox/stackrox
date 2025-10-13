package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func toV1Proto(event *storage.AdministrationEvent) *v1.AdministrationEvent {
	return &v1.AdministrationEvent{
		Id:      event.GetId(),
		Type:    toV1TypeEnum(event.GetType()),
		Level:   toV1LevelEnum(event.GetLevel()),
		Message: event.GetMessage(),
		Hint:    event.GetHint(),
		Domain:  event.GetDomain(),
		Resource: &v1.AdministrationEvent_Resource{
			Type: event.GetResource().GetType(),
			Id:   event.GetResource().GetId(),
			Name: event.GetResource().GetName(),
		},
		NumOccurrences: event.GetNumOccurrences(),
		LastOccurredAt: event.GetLastOccurredAt(),
		CreatedAt:      event.GetCreatedAt(),
	}
}

func toV1TypeEnum(val storage.AdministrationEventType) v1.AdministrationEventType {
	return v1.AdministrationEventType(v1.AdministrationEventType_value[val.String()])
}

func toV1LevelEnum(val storage.AdministrationEventLevel) v1.AdministrationEventLevel {
	return v1.AdministrationEventLevel(v1.AdministrationEventLevel_value[val.String()])
}
