package service

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	singleton Service
	once      sync.Once
)

// Singleton returns the singleton.
func Singleton() Service {
	once.Do(func() {
		singleton = New()
	})
	return singleton
}
