package handler

import (
	"net/http"

	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton http.Handler
)

// Singleton returns the singleton service handler.
func Singleton() http.Handler {
	once.Do(func() {
		singleton = New(blob.Singleton(), handlerOpts{})
	})
	return singleton
}
