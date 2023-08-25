package expiringcache

import (
	"time"
)

// Clock is an interface that provides the current time.s
//
//go:generate mockgen-wrapper
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (rc realClock) Now() time.Time {
	return time.Now()
}
