package handler

import (
	"net/http"

	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton http.Handler
)

// Singleton returns the singleton service handler.
func Singleton() http.Handler {
	once.Do(func() {
		singleton = New(fetcher.SingletonManager(), blob.Singleton(), handlerOpts{})
	})
	return singleton
}
