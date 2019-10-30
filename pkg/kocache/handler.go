package kocache

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	validPathRegex = regexp.MustCompile(`^/[a-z0-9]{64}/collector-[^/]*\.k?o\.gz$`)

	log = logging.LoggerForModule()
)

func (c *koCache) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("invalid method %s, only %s requests are supported", req.Method, http.MethodGet), http.StatusMethodNotAllowed)
		return
	}
	if !validPathRegex.MatchString(req.URL.Path) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	entry := c.GetOrAddEntry(req.URL.Path)
	if entry == nil {
		http.Error(w, "kernel object cache was shut down", http.StatusInternalServerError)
		return
	}
	defer entry.ReleaseRef()

	if !concurrency.WaitInContext(entry.DoneSig(), req.Context()) {
		http.Error(w, fmt.Sprintf("context error waiting for download from upstream: %v", req.Context().Err()), http.StatusGatewayTimeout)
		return
	}

	data, size, err := entry.Contents()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not get downloaded module contents: %v", err), http.StatusInternalServerError)
		return
	}

	hdr := w.Header()
	hdr.Set("Content-Length", strconv.FormatInt(size, 10))
	hdr.Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	r := io.NewSectionReader(data, 0, size)
	if _, err := io.Copy(w, r); err != nil {
		log.Errorf("Error serving module %s to client: %v", req.URL.Path, err)
	}
}
