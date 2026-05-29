package api

import (
	"net/http"

	scandataDS "github.com/stackrox/rox/central/scandata/datastore/singleton"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	apiHandler http.Handler
)

// Singleton returns the HTTP handler for scan data API routes.
func Singleton() http.Handler {
	once.Do(func() {
		apiHandler = Handler(scandataDS.Singleton())
	})
	return apiHandler
}
