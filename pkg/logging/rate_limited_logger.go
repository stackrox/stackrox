package logging

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

// RateLimitedLogger wraps a zap.SugaredLogger that supports rate limiting.
type RateLimitedLogger struct {
	// TODO: ROX-17312: Make sure the internals allow injection of a clock
	// and mechanisms to allow unit testing of the log consolidation and flush.
	logger    Logger
	frequency float64
	burst     int
	ticker    *time.Ticker
	stopper   concurrency.Stopper
	// TODO: ROX-17312: Use an LRU Cache with expiration and eviction here
	rateLimitedLogs *lru.Cache[string, *rateLimitedLog]
}

// NewRateLimitLogger returns a rate limited logger
//
// This function adds a timer goroutine on the stack and will break tests if used to initialize globals.
// A cleaner usage pattern would be to encapsulate the initialization and use of the global in a singleton-like function:
//
//	 var (
//		    once sync.Once
//		    logger *logging.RateLimitedLogger
//	 )
//		func getRateLimitedLogger() *logging.RateLimitedLogger {
//		    once.Do(func () {
//		        logger = newRateLimitLogger(...)
//		    })
//		    return logger
//		}
func NewRateLimitLogger(l Logger, size int, logLines int, interval time.Duration, burst int) *RateLimitedLogger {
	logCache, err := lru.NewWithEvict[string, *rateLimitedLog](size, func(key string, evictedLog *rateLimitedLog) {
		if evictedLog.count.Load() > 0 {
			evictedLog.log()
		}
	})
	if err != nil {
		l.Errorf("unable to create rate limiter cache for logger in module %q: %v", CurrentModule().name, err)
		return nil
	}
	logger := &RateLimitedLogger{
		l,
		float64(logLines) / interval.Seconds(),
		burst,
		time.NewTicker(time.Second),
		concurrency.NewStopper(),
		logCache,
	}
	runtime.SetFinalizer(logger, stopLogger)
	go logger.logFlushLoop()
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

func (rl *RateLimitedLogger) logf(level zapcore.Level, limiter string, template string, args ...interface{}) {
	payload := fmt.Sprintf(template, args...)
	var keyWriter strings.Builder
	keyWriter.WriteString(limiter)
	keyWriter.WriteString("-")
	keyWriter.WriteString(level.CapitalString())
	keyWriter.WriteString("-")
	keyWriter.WriteString(payload)
	key := keyWriter.String()
	_, _ = rl.rateLimitedLogs.ContainsOrAdd(
		key,
		newRateLimitedLog(
			rl.logger,
			level,
			rate.NewLimiter(rate.Limit(rl.frequency), rl.burst),
			limiter,
			payload,
		),
	)
	if log, ok := rl.rateLimitedLogs.Get(key); ok {
		log.count.Add(1)
		if log.rateLimiter.Allow() {
			log.log()
		}
	}
}

func (rl *RateLimitedLogger) logFlushLoop() {
	defer rl.stopper.Flow().ReportStopped()
	defer rl.ticker.Stop()
	for {
		select {
		case <-rl.ticker.C:
			rl.flush(false)
		case <-rl.stopper.Flow().StopRequested():
			rl.flush(true)
			return
		}
	}
}

func (rl *RateLimitedLogger) flush(force bool) {
	keys := rl.rateLimitedLogs.Keys()
	for _, k := range keys {
		trace, found := rl.rateLimitedLogs.Peek(k)
		if !found {
			continue
		}
		if trace.count.Load() > 0 {
			if force || trace.rateLimiter.Tokens() > 0.5 {
				// One log could be issued
				trace.log()
			}
		}
	}
}

func (rl *RateLimitedLogger) stop() {
	rl.stopper.Client().Stop()
	_ = rl.stopper.Client().Stopped().Wait()
}

func stopLogger(logger *RateLimitedLogger) {
	logger.stop()
}

const (
	limitedLogSuffixFormat = " - %d log occurrences in the last %0.1f seconds for limiter %q"
)

type rateLimitedLog struct {
	logger Logger
	// TODO: ROX-17312: Use log deadline and counter rather than rate limiter
	// There is no use-case for log bursts
	rateLimiter *rate.Limiter
	level       zapcore.Level
	last        time.Time
	limiter     string
	payload     string
	count       atomic.Int32
	logMutex    sync.Mutex
}

func newRateLimitedLog(
	logger Logger,
	level zapcore.Level,
	rateLimiter *rate.Limiter,
	limiter string,
	payload string,
) *rateLimitedLog {
	// TODO: ROX-17312: Use a single rate-limited logger for all logs.
	// Check how the logger module can be integrated in the logs.
	return &rateLimitedLog{
		logger:      logger,
		rateLimiter: rateLimiter,
		level:       level,
		// last sticks to default so the first log can be issued unaltered.
		limiter: limiter,
		payload: payload,
	}
}

func (l *rateLimitedLog) log() {
	if l.count.Load() <= 0 {
		return
	}
	l.logMutex.Lock()
	defer l.logMutex.Unlock()
	now := time.Now()
	count := l.count.Swap(0)
	var suffix string
	if !l.last.IsZero() && count > 1 {
		delta := now.Sub(l.last)
		suffix = fmt.Sprintf(
			limitedLogSuffixFormat,
			count,
			delta.Seconds(),
			l.limiter,
		)
	}
	l.logger.Logf(l.level, "%s%s", l.payload, suffix)
	l.last = now
}
