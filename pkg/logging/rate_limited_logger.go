package logging

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/lru"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap/zapcore"
)

// RateLimitedLogger wraps a zap.SugaredLogger that supports rate limiting.
type RateLimitedLogger struct {
	logger          Logger
	rateLimitedLogs lru.Cache[string, *rateLimitedLog]
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
		commonLogger = newRateLimitLogger(
			createBasicLogger(),
			cacheSize,
			rateLimitPeriod,
		)
	})
	return commonLogger
}

var (
	onEvict = func(key string, evictedLog *rateLimitedLog) {
		if evictedLog.count.Load() > 0 {
			evictedLog.log()
		}
	}
)

func newRateLimitLogger(l Logger, size int, ttl time.Duration) *RateLimitedLogger {
	if size < 0 {
		size = 0
	}
	logCache := lru.NewExpirableLRU[string, *rateLimitedLog](size, onEvict, ttl)
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
	filePathPrefix = "github.com/stackrox/stackrox/"
)

func getTrimmedFilePath(path string) string {
	prefixToCut := strings.Index(path, filePathPrefix)
	cutpath := path[prefixToCut:]
	return strings.TrimPrefix(cutpath, filePathPrefix)
}

func (rl *RateLimitedLogger) logf(level zapcore.Level, limiter string, template string, args ...interface{}) {
	payload := fmt.Sprintf(template, args...)
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file, line = "", 0
	}
	file = getTrimmedFilePath(file)
	key := getLogKey(limiter, level, file, line, payload)
	if throttledLog, found := rl.rateLimitedLogs.Get(key); found {
		throttledLog.count.Add(1)
	} else {
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
}

func (rl *RateLimitedLogger) stop() {
	// Flush logs
	rl.rateLimitedLogs.Purge()
	// Stop background thread
	rl.rateLimitedLogs.Close()
}

func stopLogger(logger *RateLimitedLogger) {
	logger.stop()
}

const (
	limitedLogSuffixFormat = " - %d log suppressed for limiter %q"
)

type rateLimitedLog struct {
	logger   Logger
	level    zapcore.Level
	limiter  string
	payload  string
	file     string
	line     int
	count    atomic.Int32
	logMutex sync.Mutex
}

func newRateLimitedLog(
	logger Logger,
	level zapcore.Level,
	limiter string,
	payload string,
	file string,
	line int,
) *rateLimitedLog {
	return &rateLimitedLog{
		logger:  logger,
		level:   level,
		limiter: limiter,
		payload: payload,
		file:    file,
		line:    line,
	}
}

func (l *rateLimitedLog) log() {
	l.logMutex.Lock()
	defer l.logMutex.Unlock()
	count := l.count.Swap(0)
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
