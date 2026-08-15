package logging

import (
	"sync"
	"sync/atomic"

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
// Use this in cases when subsequent calls should not result in new log messages, i.e. to prevent log spam.
func LogOncef(logger Logger, level zapcore.Level, template string, args ...any) {
	LogOncePerKeyf("", logger, level, template, args...)
}

// LogOncePerKeyf logs a message only once per template string (before formatting message) and provided arbitrary key.
// Use this in cases when subsequent calls should not result in new log messages, i.e. to prevent log spam.
func LogOncePerKeyf(key string, logger Logger, level zapcore.Level, template string, args ...any) {
	fullKey := template
	if key != "" {
		fullKey = key + "\x00" + template
	}

	_, seen := logOnceSeen.LoadOrStore(fullKey, nil)
	if !seen {
		logger.Logf(level, template, args...)

		if logOnceMemoryUsed.Add(1) > maxLogOnceMemory {
			logOnceSeen.Range(func(key, _ any) bool {
				logOnceSeen.Delete(key)
				return false
			})
			logOnceMemoryUsed.Add(-1)
		}
	}
}
