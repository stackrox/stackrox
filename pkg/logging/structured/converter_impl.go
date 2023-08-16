package structured

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifications"
	"go.uber.org/zap"
)

var (
	_ notifications.LogConverter = (*zapConverter)(nil)
)

type zapConverter struct{}

func (z *zapConverter) Convert(msg string, module string, context ...interface{}) *storage.Notification {
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
			if isResourceTypeField, resource := IsResourceTypeField(field); isResourceTypeField {
				resourceType = resource
				resourceTypeKey = field.Key
			}
		}
	}

	notification := &storage.Notification{
		Message:     msg,
		CreatedAt:   timestamp.TimestampNow(),
		Occurrences: 1,
	}

	if resourceType != "" {
		notification.ResourceType = resourceType
		notification.ResourceId = enc.m[resourceTypeKey]
	}

	notification.Area = notifications.GetAreaFromModule(module)
	notification.Hint = notifications.GetRemediation(notification.GetArea(), resourceType)

	return notification
}
