package logging

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConvert(t *testing.T) {
	zc := &zapLogConverter{consoleEncoder: zapcore.NewConsoleEncoder(config.EncoderConfig)}

	expectedEvent := &events.AdministrationEvent{
		Domain:       "Image Scanning",
		Hint:         events.GetHint("Image Scanning", "Image", ""),
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
		Message:      `Warn: this is an events test {"image": "some-image", "another": true}`,
		ResourceType: "Image",
		ResourceName: "some-image",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
	}

	event := zc.Convert("Warn: this is an events test", "warn", "reprocessor", ImageName("some-image"),
		zap.Bool("another", true))

	assert.Equal(t, expectedEvent.GetDomain(), event.GetDomain())
	assert.Equal(t, expectedEvent.GetHint(), event.GetHint())
	assert.Equal(t, expectedEvent.GetLevel(), event.GetLevel())
	assert.Equal(t, expectedEvent.GetMessage(), event.GetMessage())
	assert.Equal(t, expectedEvent.GetResourceID(), event.GetResourceID())
	assert.Equal(t, expectedEvent.GetResourceName(), event.GetResourceName())
	assert.Equal(t, expectedEvent.GetResourceType(), event.GetResourceType())
	assert.Equal(t, expectedEvent.GetType(), event.GetType())
	assert.NoError(t, event.Validate())
}
