package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func toV1Proto(event *storage.AdministrationEvent) *v1.AdministrationEvent {
	ar := &v1.AdministrationEvent_Resource{}
	ar.SetType(event.GetResource().GetType())
	ar.SetId(event.GetResource().GetId())
	ar.SetName(event.GetResource().GetName())
	ae := &v1.AdministrationEvent{}
	ae.SetId(event.GetId())
	ae.SetType(toV1TypeEnum(event.GetType()))
	ae.SetLevel(toV1LevelEnum(event.GetLevel()))
	ae.SetMessage(event.GetMessage())
	ae.SetHint(event.GetHint())
	ae.SetDomain(event.GetDomain())
	ae.SetResource(ar)
	ae.SetNumOccurrences(event.GetNumOccurrences())
	ae.SetLastOccurredAt(event.GetLastOccurredAt())
	ae.SetCreatedAt(event.GetCreatedAt())
	return ae
}

func toV1TypeEnum(val storage.AdministrationEventType) v1.AdministrationEventType {
	return v1.AdministrationEventType(v1.AdministrationEventType_value[val.String()])
}

func toV1LevelEnum(val storage.AdministrationEventLevel) v1.AdministrationEventLevel {
	return v1.AdministrationEventLevel(v1.AdministrationEventLevel_value[val.String()])
}
