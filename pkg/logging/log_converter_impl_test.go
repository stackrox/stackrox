package logging

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConvert(t *testing.T) {
	zc := &zapLogConverter{}

	expectedEvent := &storage.AdministrationEvent{
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
		Message:        "this is an events test",
		Hint:           events.GetHint("Image Scanning", "Image"),
		Domain:         "Image Scanning",
		ResourceType:   "Image",
		ResourceId:     "some-image",
		NumOccurrences: 1,
	}

	event := zc.Convert("this is an events test", "warn", "reprocessor", ImageName("some-image"),
		zap.Bool("another", true))

	assert.Equal(t, expectedEvent.GetType(), event.GetType())
	assert.Equal(t, expectedEvent.GetLevel(), event.GetLevel())
	assert.Equal(t, expectedEvent.GetMessage(), event.GetMessage())
	assert.Equal(t, expectedEvent.GetHint(), event.GetHint())
	assert.Equal(t, expectedEvent.GetDomain(), event.GetDomain())
	assert.Equal(t, expectedEvent.GetResourceType(), event.GetResourceType())
	assert.Equal(t, expectedEvent.GetResourceId(), event.GetResourceId())
	assert.Equal(t, expectedEvent.GetNumOccurrences(), event.GetNumOccurrences())
	assert.NotEmpty(t, event.GetLastOccurredAt())
	assert.NotEmpty(t, event.GetCreatedAt())
	assert.Empty(t, event.GetId())
}
