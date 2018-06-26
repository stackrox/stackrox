package handler

import (
	"net/http"
	"sync"

	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	"bitbucket.org/stack-rox/apollo/central/logimbue/store"
)

var (
	once sync.Once

	storage store.Store

	ha http.Handler
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	ha = New(storage)
}

// Singleton returns the HTTP handler to use to serve HTTP requests.
func Singleton() http.Handler {
	once.Do(initialize)
	return ha
}
