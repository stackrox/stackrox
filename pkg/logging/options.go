package logging

import "github.com/stackrox/rox/pkg/notifications"

// options for the logger.
type options struct {
	notificationStream    notifications.Stream
	notificationConverter notifications.LogConverter
}

// OptionsFunc allows setting log options for a logger.
type OptionsFunc = func(option *options)

// EnableNotifications enables the logger to send log statements of
// Errorw and Warnw as notifications to the end-user.
//
// Before enabling logging for your package, ensure that:
// * your module resolves to a specific domain (see pkg/notifications/domain.go).
// * notifications emitted from your specific package have hints defined to help users (see pkg/notifications/hints.go).
func EnableNotifications() OptionsFunc {
	return func(option *options) {
		option.notificationConverter = &zapLogConverter{}
		option.notificationStream = notifications.Singleton()
	}
}
