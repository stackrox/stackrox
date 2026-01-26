package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/quay/claircore/rhel"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/utils"
)

// RepositoryToCPEFetcher fetches repository-to-CPE mapping data from an upstream URL.
// It acts as a simple proxy with no caching - each call fetches from upstream.
type RepositoryToCPEFetcher struct {
	url    string
	client *http.Client
}

// NewRepositoryToCPEFetcher creates a new fetcher for the repository-to-CPE mapping.
func NewRepositoryToCPEFetcher(client *http.Client, url string) *RepositoryToCPEFetcher {
	if url == "" {
		url = rhel.DefaultRepo2CPEMappingURL
	}
	return &RepositoryToCPEFetcher{
		url:    url,
		client: client,
	}
}

// FetchResult contains the result of a Fetch operation.
type FetchResult struct {
	// Modified is true if the data has been modified since the ifModifiedSince time.
	Modified bool
	// LastModified is the timestamp to use for the next conditional request.
	LastModified string
	// Data is the mapping file (nil if Modified is false).
	Data *repositorytocpe.MappingFile
}

// Fetch retrieves the repository-to-CPE mapping from the upstream URL.
// If ifModifiedSince is non-empty, it's passed as an If-Modified-Since header.
func (f *RepositoryToCPEFetcher) Fetch(ctx context.Context, ifModifiedSince string) (*FetchResult, error) {
	slog.DebugContext(ctx, "fetching repo-to-CPE mapping from upstream", "url", f.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if ifModifiedSince != "" {
		req.Header.Set("If-Modified-Since", ifModifiedSince)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching from upstream: %w", err)
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusOK:
		// New data available.
	case http.StatusNotModified:
		slog.DebugContext(ctx, "repo-to-CPE mapping not modified")
		return &FetchResult{
			Modified:     false,
			LastModified: ifModifiedSince,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var mf repositorytocpe.MappingFile
	if err := json.NewDecoder(resp.Body).Decode(&mf); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	slog.InfoContext(ctx, "fetched repo-to-CPE mapping from upstream", "entries", len(mf.Data))
	return &FetchResult{
		Modified:     true,
		LastModified: resp.Header.Get("Last-Modified"),
		Data:         &mf,
	}, nil
}
