package logging

import (
	"sync/atomic"

	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap/zapcore"
)

// maxLogOnceMemory sets the maximum number of unique entries to track.
// When the code exceeds this amount, some previously logOnceSeen entries will be randomly dropped from tracking and so
// subsequent calls to LogOnce* will result in messages appearing in the log.
const maxLogOnceMemory = 10_000

var (
	logOnceSeen       sync.Map
	logOnceMemoryUsed atomic.Int64
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
// (capped by maxLogOnceMemory in a somewhat relaxed manner). If you want to prevent repeated varied messages
// considering args, LogOncef and LogOncePerKeyf are not for you, and you should look at RateLimitedLogger or invent
// something else.
func LogOncef(logger Logger, level zapcore.Level, template string, args ...any) {
	LogOncePerKeyf("", logger, level, template, args...)
}

// LogOncePerKeyf logs a message only once per template string (before formatting message) and provided arbitrary key.
//
// The combination of the key and the template string would be the thing which prevents logging the same message
// multiple times. Use this function when you want to log once for a certain object that can be identified by the key.
// Make sure you understand when to use and not to use this function - read doc/comment for LogOncef.
func LogOncePerKeyf(key string, logger Logger, level zapcore.Level, template string, args ...any) {
	fullKey := template
	if key != "" {
		fullKey = key + "\x00" + template
	}

	_, seen := logOnceSeen.LoadOrStore(fullKey, nil)
	if !seen {
		logger.Logf(level, template, args...)

		if logOnceMemoryUsed.Add(1) > maxLogOnceMemory {
			// This is a best-effort deletion. It's possible that multiple threads try to delete the same item but only
			// one succeeds, thought both decrement the counter. Trying to do this consistently would require more
			// heavy-weight synchronization or data structures which defeats the idea of this log-once functionality
			// being computationally cheap. We should try to avoid filling up all memory to the limit anyway.
			logOnceSeen.Range(func(randomKey, _ any) bool {
				logOnceSeen.Delete(randomKey)
				return false
			})
			logOnceMemoryUsed.Add(-1)
		}
	}
}
