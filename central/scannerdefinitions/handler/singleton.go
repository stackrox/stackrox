package handler

import (
	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton *httpHandler
)

// Singleton returns the singleton service handler.
func Singleton() *httpHandler {
	once.Do(func() {
		singleton = newHandler(blob.Singleton(), handlerOpts{})
	})
	return singleton
}
