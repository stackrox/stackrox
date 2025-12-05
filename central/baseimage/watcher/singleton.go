package watcher

import (
	"github.com/stackrox/rox/central/baseimage/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	watcher Watcher
)

// Singleton returns the global base image watcher instance.
func Singleton() Watcher {
	once.Do(func() {
		watcher = New(datastore.Singleton())
	})
	return watcher
}
