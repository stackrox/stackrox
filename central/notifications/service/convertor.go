package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func toV1Proto(notification *storage.Notification) *v1.Notification {
	return &v1.Notification{
		Id:             notification.GetId(),
		Type:           toV1TypeEnum(notification.GetType()),
		Level:          toV1LevelEnum(notification.GetLevel()),
		Message:        notification.GetMessage(),
		Hint:           notification.GetHint(),
		Domain:         notification.GetDomain(),
		ResourceType:   notification.GetResourceType(),
		ResourceId:     notification.GetResourceId(),
		NumOccurrences: notification.GetNumOccurrences(),
		LastOccurredAt: notification.GetLastOccurredAt(),
		CreatedAt:      notification.GetCreatedAt(),
	}
}

func toV1TypeEnum(val storage.NotificationType) v1.NotificationType {
	return v1.NotificationType(v1.NotificationType_value[val.String()])
}

func toV1LevelEnum(val storage.NotificationLevel) v1.NotificationLevel {
	return v1.NotificationLevel(v1.NotificationLevel_value[val.String()])
}
