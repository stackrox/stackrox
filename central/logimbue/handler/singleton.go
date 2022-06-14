package handler

import (
	"net/http"

	"github.com/stackrox/stackrox/central/logimbue/store"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	storage store.Store

	ha http.Handler
)

func initialize() {
	storage = store.Singleton()
	ha = New(storage)
}

// Singleton returns the HTTP handler to use to serve HTTP requests.
func Singleton() http.Handler {
	once.Do(initialize)
	return ha
}
