package env

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

var (
	APITokenExpirationNotificationEnabled  = RegisterBooleanSetting("ROX_TOKEN_EXPIRATION_NOTIFICATION_ENABLED", false)
	APITokenExpirationNotificationInterval = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_INTERVAL" /* default: 1 hour */, 1*time.Hour)
	APITokenExpirationStaleNotificationAge = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_BACKOFF_INTERVAL" /* default: 1 day  */, timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationWindow     = registerDurationSetting("ROX_TOKEN_EXPIRATION_DETECTION_WINDOW" /* default: 1 week */, timeutil.DaysInWeek*timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationSlice      = registerDurationSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION" /* default: 1 day */, timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationSliceName  = RegisterSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION_NAME", WithDefault("day"))
)
