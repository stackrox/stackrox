package logging

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralevents"
	"go.uber.org/zap"
)

var _ centralevents.LogConverter = (*zapLogConverter)(nil)

type zapLogConverter struct{}

func (z *zapLogConverter) Convert(msg string, level string, module string, context ...interface{}) *storage.CentralEvent {
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

	event := &storage.CentralEvent{
		Message:        msg,
		Type:           storage.CentralEventType_CENTRAL_EVENT_TYPE_LOG_MESSAGE,
		CreatedAt:      timestamp.TimestampNow(),
		LastOccurredAt: timestamp.TimestampNow(),
		NumOccurrences: 1,
		Level:          logLevelToEventLevel(level),
	}

	if resourceType != "" {
		event.ResourceType = resourceType
		event.ResourceId = enc.m[resourceTypeKey]
	}

	event.Domain = centralevents.GetDomainFromModule(module)
	event.Hint = centralevents.GetHint(event.GetDomain(), resourceType)

	return event
}

func logLevelToEventLevel(level string) storage.CentralEventLevel {
	switch level {
	case "info":
		return storage.CentralEventLevel_CENTRAL_EVENT_LEVEL_INFO
	case "warn":
		return storage.CentralEventLevel_CENTRAL_EVENT_LEVEL_WARN
	case "error":
		return storage.CentralEventLevel_CENTRAL_EVENT_LEVEL_DANGER
	default:
		return storage.CentralEventLevel_CENTRAL_EVENT_LEVEL_UNKNOWN
	}
}
