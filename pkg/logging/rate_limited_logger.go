package logging

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	pkgCacheLRU "github.com/stackrox/rox/pkg/cache/lru"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap/zapcore"
)

// RateLimitedLogger wraps a zap.SugaredLogger that supports rate limiting.
type RateLimitedLogger struct {
	logger          Logger
	rateLimitedLogs pkgCacheLRU.LRU[string, *rateLimitedLog]
}

const (
	cacheSize       = 500
	rateLimitPeriod = 10 * time.Second
)

var (
	commonLogger *RateLimitedLogger
	once         sync.Once
)

// GetRateLimitedLogger returns a reference to a unique rate limited logger
//
// This function can add a timer goroutine on the stack and will break tests if used to initialize globals.
// A cleaner usage pattern would be to call the function directly when rate-limited logging functions should be used.
//
//	logging.GetRatedLimitedLogger().ErrorL("logLimiter", "This is a rate-limited error log")
func GetRateLimitedLogger() *RateLimitedLogger {
	once.Do(func() {
		logCache := lru.NewLRU[string, *rateLimitedLog](cacheSize, onEvict, rateLimitPeriod)
		commonLogger = newRateLimitLogger(
			rootLogger,
			logCache,
		)
	})
	return commonLogger
}

var (
	onEvict = func(key string, evictedLog *rateLimitedLog) {
		if evictedLog == nil {
			return
		}
		evictedLog.log()
	}
)

func newRateLimitLogger(l Logger, logCache pkgCacheLRU.LRU[string, *rateLimitedLog]) *RateLimitedLogger {
	logger := &RateLimitedLogger{
		l,
		logCache,
	}
	runtime.SetFinalizer(logger, stopLogger)
	return logger
}

// ErrorL logs a templated error message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) ErrorL(limiter string, template string, args ...interface{}) {
	rl.logf(zapcore.ErrorLevel, limiter, template, args...)
}

// Error logs the consecutive interfaces
func (rl *RateLimitedLogger) Error(args ...interface{}) {
	rl.logger.Error(args...)
}

// Errorf logs the input template filled with argument data
func (rl *RateLimitedLogger) Errorf(template string, args ...interface{}) {
	rl.logger.Errorf(template, args...)
}

// Errorw logs the input message and keyValues
func (rl *RateLimitedLogger) Errorw(msg string, keysAndValues ...interface{}) {
	rl.logger.Errorw(msg, keysAndValues...)
}

// WarnL logs a templated warn message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) WarnL(limiter string, template string, args ...interface{}) {
	rl.logf(zapcore.WarnLevel, limiter, template, args...)
}

// Warn logs the consecutive interfaces
func (rl *RateLimitedLogger) Warn(args ...interface{}) {
	rl.logger.Warn(args...)
}

// Warnf logs the input template filled with argument data
func (rl *RateLimitedLogger) Warnf(template string, args ...interface{}) {
	rl.logger.Warnf(template, args...)
}

// Warnw logs the input message and keyValues
func (rl *RateLimitedLogger) Warnw(msg string, keysAndValues ...interface{}) {
	rl.logger.Warnw(msg, keysAndValues...)
}

// InfoL logs a templated info message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) InfoL(limiter string, template string, args ...interface{}) {
	rl.logf(zapcore.InfoLevel, limiter, template, args...)
}

// Info logs the consecutive interfaces
func (rl *RateLimitedLogger) Info(args ...interface{}) {
	rl.logger.Info(args...)
}

// Infof logs the input template filled with argument data
func (rl *RateLimitedLogger) Infof(template string, args ...interface{}) {
	rl.logger.Infof(template, args...)
}

// Infow logs the input message and keyValues
func (rl *RateLimitedLogger) Infow(msg string, keysAndValues ...interface{}) {
	rl.logger.Infow(msg, keysAndValues...)
}

// DebugL logs a templated debug message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) DebugL(limiter string, template string, args ...interface{}) {
	rl.logf(zapcore.DebugLevel, limiter, template, args...)
}

// Debug logs the consecutive interfaces
func (rl *RateLimitedLogger) Debug(args ...interface{}) {
	rl.logger.Debug(args...)
}

// Debugf logs the input template filled with argument data
func (rl *RateLimitedLogger) Debugf(template string, args ...interface{}) {
	rl.logger.Debugf(template, args...)
}

// Debugw logs the input message and keyValues
func (rl *RateLimitedLogger) Debugw(msg string, keysAndValues ...interface{}) {
	rl.logger.Debugw(msg, keysAndValues...)
}

func getLogKey(limiter string, level zapcore.Level, file string, line int, payload string) string {
	var keyWriter strings.Builder
	keyWriter.WriteString(limiter)
	keyWriter.WriteString("-")
	keyWriter.WriteString(level.CapitalString())
	keyWriter.WriteString("-")
	keyWriter.WriteString(file)
	keyWriter.WriteString(":")
	keyWriter.WriteString(fmt.Sprintf("%d", line))
	keyWriter.WriteString("-")
	keyWriter.WriteString(payload)
	return keyWriter.String()
}

const (
	localFilePathPrefix = "github.com/stackrox/stackrox/"
	filePathPrefix      = "github.com/stackrox/rox/"
	githubPathPrefix    = "/__w/stackrox/stackrox/"
)

func getTrimmedFilePath(path string) string {
	prefixes := []string{filePathPrefix, localFilePathPrefix, githubPathPrefix}
	for _, prefix := range prefixes {
		prefixToCut := strings.Index(path, prefix)
		if prefixToCut >= 0 {
			cutPrefixLength := prefixToCut + len(prefix)
			return path[cutPrefixLength:]
		}
	}
	return path
}

func (rl *RateLimitedLogger) registerTraceAndLog(
	level zapcore.Level,
	limiter string,
	key string,
	payload string,
	file string,
	line int,
) {
	log := newRateLimitedLog(
		rl.logger,
		level,
		limiter,
		payload,
		file,
		line,
	)
	rl.rateLimitedLogs.Add(key, log)
	log.log()
}

func (rl *RateLimitedLogger) logf(level zapcore.Level, limiter string, template string, args ...interface{}) {
	payload := fmt.Sprintf(template, args...)
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file, line = "", 0
	}
	file = getTrimmedFilePath(file)
	key := getLogKey(limiter, level, file, line, payload)
	if throttledLog, found := rl.rateLimitedLogs.Get(key); found && throttledLog != nil {
		throttledLog.count.Add(1)
		// In case the log were evicted between cache lookup and count increase,
		// check for existence in the cache after the increase, and log if the retrieved
		// log is not in the cache anymore.
		if checkLog, checkFound := rl.rateLimitedLogs.Get(key); !checkFound || throttledLog.getID() != checkLog.getID() {
			throttledLog.log()
		}
	} else if found && throttledLog == nil {
		// There is something wrong in the cache. Clean up.
		rl.rateLimitedLogs.Remove(key)
		rl.registerTraceAndLog(level, limiter, key, payload, file, line)
	} else {
		rl.registerTraceAndLog(level, limiter, key, payload, file, line)
	}
}

func (rl *RateLimitedLogger) stop() {
	// Flush logs
	rl.rateLimitedLogs.Purge()
}

func stopLogger(logger *RateLimitedLogger) {
	logger.stop()
}

const (
	limitedLogSuffixFormat = " - %d log suppressed for limiter %q"
)

type rateLimitedLog struct {
	logger  Logger
	id      string
	level   zapcore.Level
	limiter string
	payload string
	file    string
	line    int
	count   atomic.Int32
}

func newRateLimitedLog(
	logger Logger,
	level zapcore.Level,
	limiter string,
	payload string,
	file string,
	line int,
) *rateLimitedLog {
	log := &rateLimitedLog{
		logger:  logger,
		id:      uuid.NewV4().String(),
		level:   level,
		limiter: limiter,
		payload: payload,
		file:    file,
		line:    line,
	}
	log.count.Add(1)
	return log
}

func (l *rateLimitedLog) getID() string {
	if l == nil {
		return ""
	}
	return l.id
}

func (l *rateLimitedLog) log() {
	count := l.count.Swap(0)
	if count == 0 {
		return
	}
	var suffix string
	if count > 1 {
		suffix = fmt.Sprintf(
			limitedLogSuffixFormat,
			count,
			l.limiter,
		)
	}
	var prefix string
	if len(l.file) > 0 && l.line > 0 {
		prefix = fmt.Sprintf("%s:%d - ", l.file, l.line)
	}
	l.logger.Logf(l.level, "%s%s%s", prefix, l.payload, suffix)
}
