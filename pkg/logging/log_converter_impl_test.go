package logging

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConvert(t *testing.T) {
	zc := &zapLogConverter{consoleEncoder: zapcore.NewConsoleEncoder(config.EncoderConfig)}

	cases := []struct {
		event  *events.AdministrationEvent
		msg    string
		level  string
		module string
		fields []interface{}
	}{
		{
			event: &events.AdministrationEvent{
				Domain:       "Image Scanning",
				Hint:         events.GetHint("Image Scanning", "Image", ""),
				Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
				Message:      `Warn: this is an events test {"image": "some-image", "another": true}`,
				ResourceType: "Image",
				ResourceName: "some-image",
				Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
			},
			msg:    "Warn: this is an events test",
			level:  "warn",
			module: "reprocessor",
			fields: []interface{}{ImageName("some-image"), zap.Bool("another", true)},
		},
		{
			event: &events.AdministrationEvent{
				Domain:       "Integrations",
				Hint:         events.GetHint("Integrations", "Notifier", "awssh-cache-exhausted"),
				Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
				Message:      `Error: this is an events test {"notifier": "some-notifier", "something": "somewhere", "err_code": "awssh-cache-exhausted"}`,
				ResourceType: "Notifier",
				ResourceName: "some-notifier",
				Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
			},
			msg:    "Error: this is an events test",
			level:  "error",
			module: "pkg/notifiers/awssh",
			fields: []interface{}{
				NotifierName("some-notifier"), String("something", "somewhere"),
				ErrCode("awssh-cache-exhausted"),
			},
		},
		{
			event: &events.AdministrationEvent{
				Domain:       "Authentication",
				Hint:         events.GetHint("Authentication", "API Token", ""),
				Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
				Message:      `Error: this is an events test {"api_token_id": "some-token-id", "api_token_name": "some-token-name"}`,
				ResourceType: "API Token",
				ResourceName: "some-token-name",
				ResourceID:   "some-token-id",
				Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
			},
			msg:    "Error: this is an events test",
			level:  "error",
			module: "apitoken/expiration",
			fields: []interface{}{
				APITokenID("some-token-id"), APITokenName("some-token-name"),
			},
		},
		{
			msg:    "Error: something went wrong",
			level:  "error",
			module: "pkg/random/something",
			fields: []interface{}{
				String("response", "some api response"),
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("convert event %d", i), func(t *testing.T) {
			event := zc.Convert(tc.msg, tc.level, tc.module, tc.fields...)
			assert.Equal(t, tc.event.GetDomain(), event.GetDomain())
			assert.Equal(t, tc.event.GetHint(), event.GetHint())
			assert.Equal(t, tc.event.GetLevel(), event.GetLevel())
			assert.Equal(t, tc.event.GetMessage(), event.GetMessage())
			assert.Equal(t, tc.event.GetResourceID(), event.GetResourceID())
			assert.Equal(t, tc.event.GetResourceName(), event.GetResourceName())
			assert.Equal(t, tc.event.GetResourceType(), event.GetResourceType())
			assert.Equal(t, tc.event.GetType(), event.GetType())
			if tc.event != nil {
				assert.NoError(t, event.Validate())
			}
		})
	}
}
