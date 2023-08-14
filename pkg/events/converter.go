package events

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap"
)

// LogConverter converts a log entry to a storage.Event
type LogConverter interface {
	Convert(msg string, context ...interface{}) *storage.Event
}

type zapConverter struct{}

func (z *zapConverter) Convert(msg string, context ...interface{}) *storage.Event {
	enc := &stringObjectEncoder{
		m: make(map[string]string, len(context)),
	}

	// For now, the assumption is that structured logging with our current logger uses the construct
	// according to https://github.com/uber-go/zap/blob/master/field.go. Thus, the given interfaces
	// shall be a strongly-typed zap.Field.
	for _, c := range context {
		// Currently silently drop the given context of the log entry if it's not a zap.Field.
		if field, ok := c.(zap.Field); ok {
			field.AddTo(enc)
		}
	}

	return &storage.Event{
		Id:        uuid.NewV4().String(),
		Message:   msg,
		Labels:    enc.m,
		CreatedAt: timestamp.TimestampNow(),
	}
}
