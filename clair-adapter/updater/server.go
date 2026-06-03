package updater

import (
	"net/http"
	"strings"
	"sync"
)

// Server serves vulnerability data to Clair.
// Each bundle is served at its own endpoint (/updater/{name} or /enricher/{name}).
type Server struct {
	mu      sync.RWMutex
	bundles map[string]*BundleData // keyed by bundle name
	mux     *http.ServeMux
}

// NewServer creates a new updater HTTP server.
func NewServer() *Server {
	s := &Server{
		bundles: make(map[string]*BundleData),
		mux:     http.NewServeMux(),
	}

	// Register catch-all handler
	s.mux.HandleFunc("/", s.handleBundle)

	return s
}

// SetBundles atomically updates the available bundles.
func (s *Server) SetBundles(bundles []*BundleData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing bundles
	clear(s.bundles)

	// Add new bundles
	for _, bundle := range bundles {
		s.bundles[bundle.Name] = bundle
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleBundle handles requests for vulnerability bundles.
func (s *Server) handleBundle(w http.ResponseWriter, r *http.Request) {
	// Parse path: /updater/{name} or /enricher/{name}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) != 2 || (parts[0] != "updater" && parts[0] != "enricher") {
		http.NotFound(w, r)
		return
	}

	bundleName := parts[1]

	s.mu.RLock()
	bundle, ok := s.bundles[bundleName]
	s.mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	// Check If-None-Match for conditional GET
	if r.Header.Get("If-None-Match") == bundle.Fingerprint {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set headers
	w.Header().Set("ETag", bundle.Fingerprint)
	w.Header().Set("Content-Type", "application/zstd")

	// Write data
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bundle.Data)
}
