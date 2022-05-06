package handler

import (
	"net/http"

	"github.com/stackrox/rox/central/cve/fetcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton ScannerDefinitionsHandler
)

// ScannerDefinitionsHandler is an http.Handler that also has information on how to get vuln definitions info.
type ScannerDefinitionsHandler interface {
	http.Handler
	GetVulnDefsInfo() (*v1.VulnDefinitionsInfo, error)
}

// Singleton returns the singleton service handler.
func Singleton() ScannerDefinitionsHandler {
	once.Do(func() {
		singleton = New(fetcher.SingletonManager(), handlerOpts{})
	})
	return singleton
}
