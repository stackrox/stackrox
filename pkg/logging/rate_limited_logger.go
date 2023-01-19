package logging

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/time/rate"
)

// RateLimitedLogger wraps a zap.SugaredLogger that supports rate limiting.
type RateLimitedLogger struct {
	*Logger
	frequency    float64
	burst        int
	rateLimiters *lru.Cache
}

// NewRateLimitLogger returns a rate limited logger
func NewRateLimitLogger(l *Logger, size int, logLines int, interval time.Duration, burst int) *RateLimitedLogger {
	cache, err := lru.New(size)
	if err != nil {
		l.Errorf("unable to create rate limiter cache for logger in module %q: %v", l.module.name, err)
		return nil
	}
	return &RateLimitedLogger{
		l,
		float64(logLines) / interval.Seconds(),
		burst,
		cache,
	}
}

// ErrorL logs a templated error message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) ErrorL(limiter string, template string, args ...interface{}) {
	if rl.allowLog(limiter) {
		rl.Errorf(template, args...)
	}
}

// WarnL logs a templated warn message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) WarnL(limiter string, template string, args ...interface{}) {
	if rl.allowLog(limiter) {
		rl.Warnf(template, args...)
	}
}

// InfoL logs a templated info message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) InfoL(limiter string, template string, args ...interface{}) {
	if rl.allowLog(limiter) {
		rl.Infof(template, args...)
	}
}

// DebugL logs a templated debug message if allowed by the rate limiter corresponding to the identifier
func (rl *RateLimitedLogger) DebugL(limiter string, template string, args ...interface{}) {
	if rl.allowLog(limiter) {
		rl.Debugf(template, args...)
	}
}

func (rl *RateLimitedLogger) allowLog(limiter string) bool {
	_, _ = rl.rateLimiters.ContainsOrAdd(limiter, rate.NewLimiter(rate.Limit(rl.frequency), rl.burst))

	if lim, ok := rl.rateLimiters.Get(limiter); ok {
		return lim.(*rate.Limiter).Allow()
	}
	return false
}
