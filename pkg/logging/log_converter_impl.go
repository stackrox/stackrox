package logging

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/buildinfo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ events.LogConverter = (*zapLogConverter)(nil)

type zapLogConverter struct {
	consoleEncoder zapcore.Encoder
}

func (z *zapLogConverter) Convert(msg string, level string, module string, context ...interface{}) *events.AdministrationEvent {
	enc := &stringObjectEncoder{
		m:              make(map[string]string, len(context)),
		consoleEncoder: z.consoleEncoder,
	}
	fields := make([]zap.Field, 0, len(context))

	// For now, the assumption is that structured logging with our current logger uses the construct
	// according to https://github.com/uber-go/zap/blob/master/field.go. Thus, the given interfaces
	// shall be a strongly-typed zap.Field.
	var resourceType string
	var resourceTypeKey string
	for _, c := range context {
		// Currently silently drop the given context of the log entry if it's not a zap.Field.
		if field, ok := c.(zap.Field); ok {
			field.AddTo(enc)
			fields = append(fields, field)
			if resource, exists := getResourceTypeField(field); exists {
				if resourceType != "" {
					// We cannot import utils.Should, hence need to handle this conditionally here ourselves.
					err := fmt.Errorf("duplicate resource field found: %s", field.Key)
					should(err)
				} else {
					resourceType = resource
					resourceTypeKey = field.Key
				}
			}
		}
	}

	event := &events.AdministrationEvent{
		Type:  storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		Level: logLevelToEventLevel(level),
	}

	msgWithContext, err := enc.CreateMessage(msg, level, fields)
	if err != nil {
		should(err)
	}

	event.Message = msgWithContext

	if resourceType != "" {
		event.ResourceType = resourceType
		if isIDField(resourceTypeKey) {
			event.ResourceID = enc.m[resourceTypeKey]
		} else {
			event.ResourceName = enc.m[resourceTypeKey]
		}
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

func should(err error) {
	if buildinfo.ReleaseBuild {
		thisModuleLogger.Errorf("Failed to create event: %v", err)
	} else {
		thisModuleLogger.Panicf("Failed to create event: %v", err)
	}
}
