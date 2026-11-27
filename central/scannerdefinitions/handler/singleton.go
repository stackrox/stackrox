package handler

import (
	"context"
	"net/http"

	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton *httpHandler
)

// Singleton returns the singleton service handler.
func Singleton() http.Handler {
	return getSingleton()
}

// RepositoryToCPEMapping loads the mapping served to connected scanners. The
// returned mapping is shared and callers must treat it as read-only.
func RepositoryToCPEMapping(ctx context.Context) (*repositorytocpe.MappingFile, error) {
	return getSingleton().repositoryToCPEMapping(ctx)
}

func getSingleton() *httpHandler {
	once.Do(func() {
		singleton = newHandler(blob.Singleton(), handlerOpts{})
	})
	return singleton
}
