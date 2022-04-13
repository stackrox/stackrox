package handler

import (
	"net/http"

	"github.com/stackrox/stackrox/central/cve/fetcher"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once      sync.Once
	singleton http.Handler
)

// Singleton returns the singleton service handler.
func Singleton() http.Handler {
	once.Do(func() {
		singleton = New(fetcher.SingletonManager(), handlerOpts{})
	})
	return singleton
}
