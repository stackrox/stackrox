package env

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

var (
	APITokenExpirationNotificationInterval = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_INTERVAL" /* 1 hour */, 1*time.Hour)
	APITokenExpirationStaleNotificationAge = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_BACKOFF_INTERVAL" /* 1 day  */, timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationWindow     = registerDurationSetting("ROX_TOKEN_EXPIRATION_DETECTION_WINDOW" /* 1 week */, timeutil.DaysInWeek*timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationSlice      = registerDurationSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION" /* 1 day */, timeutil.HoursInDay*time.Hour)
	APITokenExpirationExpirationSliceName  = RegisterSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION_NAME", WithDefault("day"))
)
