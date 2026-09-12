package logging

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap/zapcore"
)

// maxLogOnceMemory sets the maximum number of unique entries to track.
// When the use exceeds this amount, some previously logOnceSeen entries will be randomly dropped and so subsequent
// calls to LogOnce* will result in previously seen messages again appearing in the log.
// If we see a warning in the logs (ref logOnceLimitNotified) that we reached this limit, we should check our use of
// LogOncef / LogOncePerKeyf and remove any cases when the same line sends varying templates, or bump the limit if
// there are no such cases.
const maxLogOnceMemory = 10_000

var (
	logOnceSeen          sync.Map
	logOnceMemoryUsed    atomic.Int64
	logOnceLimitNotified atomic.Bool
)

// LogOncef logs a message only once per template string (before formatting message).
//
// Use this in cases when subsequent calls should not result in new log messages, i.e. to prevent log spam.
// The idea is that you can call LogOncef in some code that may get called repeatedly and should log something useful,
// but the conditions don't change. You would use this function if you want to avoid producing the same repeated
// message to logs over and over.
// It is important that repeated messages are prevented for the same template string which is used before formatting
// args into it. This is intentional compromise: LogOncef and LogOncePerKeyf are more performant than RateLimitedLogger
// because they don't rely on heavy synchronization with Mutex and use relatively small memory of seen messages
// (capped by maxLogOnceMemory). If you want to prevent repeated varied messages considering args, LogOncef and
// LogOncePerKeyf are not for you, and you should look at RateLimitedLogger or invent something else.
// Note that level also does not participate in de-duplication (similar to args).
func LogOncef(logger Logger, level zapcore.Level, template string, args ...any) {
	LogOncePerKeyf("", logger, level, template, args...)
}

// LogOncePerKeyf logs a message only once per template string (before formatting message) and provided arbitrary key.
//
// The combination of the key and the template string would be the thing which prevents logging the same message
// multiple times. Use this function when you want to log once for a certain object that can be identified by the key.
// Make sure you understand when to use and not to use this function - read doc/comment for LogOncef.
// It's important to make sure the number of keys is bounded. For example, the use of cluster IDs is ok there as we
// know that the number of clusters is limited, but the use of container IDs is not because many new containers are
// likely to appear during the run time of the process.
func LogOncePerKeyf(key string, logger Logger, level zapcore.Level, template string, args ...any) {
	fullKey := template
	if key != "" {
		fullKey = key + "\x00" + template
	}

	_, seen := logOnceSeen.LoadOrStore(fullKey, nil)
	if !seen {
		logger.Logf(level, template, args...)

		if logOnceMemoryUsed.Add(1) > maxLogOnceMemory {
			if !logOnceLimitNotified.Swap(true) {
				logger.Warnf("maxLogOnceMemory=%d limit reached", maxLogOnceMemory)
			}
			logOnceSeen.Range(func(randomKey, _ any) bool {
				if randomKey == fullKey {
					// Don't forget what we just added, iterate to try another randomKey.
					return true
				}
				if _, deleted := logOnceSeen.LoadAndDelete(randomKey); deleted {
					logOnceMemoryUsed.Add(-1)
					return false // Stop iterating.
				}
				// Some other thread deleted the same randomKey, keep iterating to try another one.
				return true
			})
		}
	}
}
