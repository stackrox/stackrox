package logging

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifications"
	"go.uber.org/zap"
)

var (
	_ notifications.LogConverter = (*zapLogConverter)(nil)
)

type zapLogConverter struct{}

func (z *zapLogConverter) Convert(msg string, level string, module string, context ...interface{}) *storage.Notification {
	enc := &stringObjectEncoder{
		m: make(map[string]string, len(context)),
	}

	// For now, the assumption is that structured logging with our current logger uses the construct
	// according to https://github.com/uber-go/zap/blob/master/field.go. Thus, the given interfaces
	// shall be a strongly-typed zap.Field.
	var resourceType string
	var resourceTypeKey string
	for _, c := range context {
		// Currently silently drop the given context of the log entry if it's not a zap.Field.
		if field, ok := c.(zap.Field); ok {
			field.AddTo(enc)
			if resource, exists := getResourceTypeField(field); exists {
				if resourceType != "" {
					thisModuleLogger.Warnf("Received multiple resource field in structured log."+
						" Previous resource %q will be overwritten by %q", resourceType, resource)
				}
				resourceType = resource
				resourceTypeKey = field.Key
			}
		}
	}

	notification := &storage.Notification{
		Message:      msg,
		Type:         storage.NotificationType_NOTIFICATION_TYPE_LOG_MESSAGE,
		CreatedAt:    timestamp.TimestampNow(),
		LastOccurred: timestamp.TimestampNow(),
		Occurrences:  1,
		Level:        logLevelToNotificationLevel(level),
	}

	if resourceType != "" {
		notification.ResourceType = resourceType
		notification.ResourceId = enc.m[resourceTypeKey]
	}

	notification.Domain = notifications.GetDomainFromModule(module)
	notification.Hint = notifications.GetHint(notification.GetDomain(), resourceType)

	return notification
}

func logLevelToNotificationLevel(level string) storage.NotificationLevel {
	switch level {
	case "info":
		return storage.NotificationLevel_NOTIFICATION_LEVEL_INFO
	case "warn":
		return storage.NotificationLevel_NOTIFICATION_LEVEL_WARN
	case "error":
		return storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER
	default:
		return storage.NotificationLevel_NOTIFICATION_LEVEL_UNKNOWN
	}
}
