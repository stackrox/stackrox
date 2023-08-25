package logging

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifications"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConvert(t *testing.T) {
	zc := &zapLogConverter{}

	expectedNotification := &storage.Notification{
		Type:         storage.NotificationType_NOTIFICATION_TYPE_LOG_MESSAGE,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_WARN,
		Message:      "this is a notification test",
		Hint:         notifications.GetHint("Image Scanning", "Image"),
		Domain:       "Image Scanning",
		ResourceType: "Image",
		ResourceId:   "some-image",
		Occurrences:  1,
	}

	notification := zc.Convert("this is a notification test", "warn", "reprocessor", ImageName("some-image"),
		zap.Bool("another", true))

	assert.Equal(t, expectedNotification.GetType(), notification.GetType())
	assert.Equal(t, expectedNotification.GetLevel(), notification.GetLevel())
	assert.Equal(t, expectedNotification.GetMessage(), notification.GetMessage())
	assert.Equal(t, expectedNotification.GetHint(), notification.GetHint())
	assert.Equal(t, expectedNotification.GetDomain(), notification.GetDomain())
	assert.Equal(t, expectedNotification.GetResourceType(), notification.GetResourceType())
	assert.Equal(t, expectedNotification.GetResourceId(), notification.GetResourceId())
	assert.Equal(t, expectedNotification.GetOccurrences(), notification.GetOccurrences())
	assert.NotEmpty(t, notification.GetLastOccurred())
	assert.NotEmpty(t, notification.GetCreatedAt())
	assert.Empty(t, notification.GetId())
}
