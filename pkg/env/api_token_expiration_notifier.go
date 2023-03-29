package env

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

var (
	// APITokenExpirationNotificationInterval is the duration of the interval between two notification loop runs.
	APITokenExpirationNotificationInterval = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_INTERVAL" /* default: 1 hour */, 1*time.Hour)
	// APITokenExpirationStaleNotificationAge is the duration during which no new notification will be sent out.
	APITokenExpirationStaleNotificationAge = registerDurationSetting("ROX_TOKEN_EXPIRATION_NOTIFIER_BACKOFF_INTERVAL" /* default: 1 day  */, timeutil.HoursInDay*time.Hour)
	// APITokenExpirationExpirationWindow is the duration of the window taken from the current point in time during which token expiration date will trigger notification.
	APITokenExpirationExpirationWindow = registerDurationSetting("ROX_TOKEN_EXPIRATION_DETECTION_WINDOW" /* default: 1 week */, timeutil.DaysInWeek*timeutil.HoursInDay*time.Hour)
	// APITokenExpirationExpirationSlice is the duration of the slice used to generate the expiration log
	APITokenExpirationExpirationSlice = registerDurationSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION" /* default: 1 day */, timeutil.HoursInDay*time.Hour)
	// APITokenExpirationExpirationSliceName is the name used for the time slice described by APITokenExpirationExpirationSlice
	APITokenExpirationExpirationSliceName = RegisterSetting("ROX_TOKEN_EXPIRATION_LOG_SLICE_DURATION_NAME", WithDefault("day"))
)
