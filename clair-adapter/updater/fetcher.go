package updater

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Fetcher fetches vulnerability data from URLs in online mode.
type Fetcher struct {
	server       *Server
	client       *http.Client
	urls         []string      // candidate URLs, tried in order
	interval     time.Duration // default 5 minutes
	lastModified string        // for If-Modified-Since header
}

// FetcherOption configures the Fetcher.
type FetcherOption func(*Fetcher)

// WithFetchInterval sets the interval between fetch cycles.
// Default is 5 minutes.
func WithFetchInterval(d time.Duration) FetcherOption {
	return func(f *Fetcher) {
		f.interval = d
	}
}

// WithHTTPClient sets the HTTP client to use for requests.
// Default is http.DefaultClient.
func WithHTTPClient(c *http.Client) FetcherOption {
	return func(f *Fetcher) {
		f.client = c
	}
}

// NewFetcher creates a new vulnerability data fetcher.
func NewFetcher(server *Server, urls []string, opts ...FetcherOption) *Fetcher {
	f := &Fetcher{
		server:   server,
		client:   http.DefaultClient,
		urls:     urls,
		interval: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Start starts the fetch loop and blocks until the context is canceled.
// It fetches immediately on start, then runs at the configured interval.
func (f *Fetcher) Start(ctx context.Context) error {
	// Fetch immediately on start
	if err := f.FetchOnce(ctx); err != nil {
		slog.ErrorContext(ctx, "Initial fetch failed", "error", err)
	}

	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := f.FetchOnce(ctx); err != nil {
				slog.ErrorContext(ctx, "Fetch cycle failed", "error", err)
			}
		}
	}
}

// FetchOnce performs a single fetch cycle.
func (f *Fetcher) FetchOnce(ctx context.Context) error {
	if len(f.urls) == 0 {
		return fmt.Errorf("no URLs configured")
	}

	var lastErr error

	// Try each URL in order
	for _, url := range f.urls {
		slog.InfoContext(ctx, "Fetching vulnerability data", "url", url)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Add If-Modified-Since header if we have a last modified time
		if f.lastModified != "" {
			req.Header.Set("If-Modified-Since", f.lastModified)
		}

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		switch resp.StatusCode {
		case http.StatusNotModified:
			resp.Body.Close()
			slog.InfoContext(ctx, "Vulnerability data not modified")
			return nil

		case http.StatusOK:
			// Download to temporary file
			tmpFile, err := os.CreateTemp("", "vulnerabilities-*.zip")
			if err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("failed to create temp file: %w", err)
				continue
			}
			tmpPath := tmpFile.Name()

			// Copy response to temp file
			_, err = io.Copy(tmpFile, resp.Body)
			resp.Body.Close()
			tmpFile.Close()

			if err != nil {
				os.Remove(tmpPath)
				lastErr = fmt.Errorf("failed to download file: %w", err)
				continue
			}

			// Unpack bundles
			bundles, err := UnpackBundle(tmpPath)
			os.Remove(tmpPath)

			if err != nil {
				lastErr = fmt.Errorf("failed to unpack bundle: %w", err)
				continue
			}

			// Update server with new bundles
			f.server.SetBundles(bundles)

			// Update last modified time from response header
			if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
				f.lastModified = lastMod
			}

			slog.InfoContext(ctx, "Successfully fetched and loaded vulnerability data",
				"bundles", len(bundles),
				"url", url)
			return nil

		case http.StatusNotFound:
			resp.Body.Close()
			lastErr = fmt.Errorf("URL not found: %s", url)
			continue

		default:
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			continue
		}
	}

	return fmt.Errorf("all URLs failed, last error: %w", lastErr)
}
