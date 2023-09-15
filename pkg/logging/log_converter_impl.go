package logging

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"go.uber.org/zap"
)

var _ events.LogConverter = (*zapLogConverter)(nil)

type zapLogConverter struct{}

func (z *zapLogConverter) Convert(msg string, level string, module string, context ...interface{}) *events.AdministrationEvent {
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

	event := &events.AdministrationEvent{
		Message: msg,
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		Level:   logLevelToEventLevel(level),
	}

	if resourceType != "" {
		event.ResourceType = resourceType
		event.ResourceID = enc.m[resourceTypeKey]
	}

	event.Domain = events.GetDomainFromModule(module)
	event.Hint = events.GetHint(event.GetDomain(), resourceType)

	return event
}

func logLevelToEventLevel(level string) storage.AdministrationEventLevel {
	switch level {
	case "info":
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO
	case "warn":
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING
	case "error":
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR
	default:
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_UNKNOWN
	}
}
