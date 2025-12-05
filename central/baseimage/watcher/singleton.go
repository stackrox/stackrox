package watcher

import (
	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	watcher Watcher
)

// Singleton returns the global base image watcher instance.
func Singleton() Watcher {
	once.Do(func() {
		watcher = New(repoDS.Singleton())
	})
	return watcher
}
