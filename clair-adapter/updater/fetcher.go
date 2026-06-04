package updater

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/mtls"
)

// BundleImporter imports vulnerability bundles from a ZIP file on disk.
type BundleImporter interface {
	ImportFromZip(ctx context.Context, zipPath string) error
}

// Fetcher fetches vulnerability data from URLs in online mode.
type Fetcher struct {
	server       *Server
	importer     BundleImporter
	client       *http.Client
	urls         []string
	interval     time.Duration
	lastModified string
}

// FetcherOption configures the Fetcher.
type FetcherOption func(*Fetcher)

// WithFetchInterval sets the interval between fetch cycles.
func WithFetchInterval(d time.Duration) FetcherOption {
	return func(f *Fetcher) { f.interval = d }
}

// WithHTTPClient sets the HTTP client to use for requests.
func WithHTTPClient(c *http.Client) FetcherOption {
	return func(f *Fetcher) { f.client = c }
}

// WithImporter sets the bundle importer called after successful fetch.
func WithImporter(imp BundleImporter) FetcherOption {
	return func(f *Fetcher) { f.importer = imp }
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
func (f *Fetcher) Start(ctx context.Context) error {
	if err := f.FetchOnce(ctx); err != nil {
		slog.ErrorContext(ctx, "initial fetch failed", "error", err)
	}

	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := f.FetchOnce(ctx); err != nil {
				slog.ErrorContext(ctx, "fetch cycle failed", "error", err)
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

	for _, url := range f.urls {
		slog.InfoContext(ctx, "fetching vulnerability data", "url", url)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("creating request: %w", err)
			continue
		}
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
			slog.InfoContext(ctx, "vulnerability data not modified")
			return nil

		case http.StatusOK:
			tmpFile, err := os.CreateTemp("", "vulnerabilities-*.zip")
			if err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("creating temp file: %w", err)
				continue
			}
			tmpPath := tmpFile.Name()

			_, err = io.Copy(tmpFile, resp.Body)
			resp.Body.Close()
			tmpFile.Close()
			if err != nil {
				os.Remove(tmpPath)
				lastErr = fmt.Errorf("downloading: %w", err)
				continue
			}

			// Import into Clair's DB by streaming from the ZIP on disk.
			if f.importer != nil {
				if err := f.importer.ImportFromZip(ctx, tmpPath); err != nil {
					slog.ErrorContext(ctx, "failed to import bundles into Clair DB", "error", err)
				}
			}

			// Unpack bundles into memory for the HTTP server.
			bundles, err := UnpackBundle(tmpPath)
			os.Remove(tmpPath)
			if err != nil {
				lastErr = fmt.Errorf("unpacking bundle: %w", err)
				continue
			}

			f.server.SetBundles(bundles)

			if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
				f.lastModified = lastMod
			}

			slog.InfoContext(ctx, "vulnerability data loaded",
				"bundles", len(bundles), "url", url, "imported", f.importer != nil)
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

// NewMTLSHTTPClient creates an HTTP client configured with StackRox mTLS
// certificates for authenticating to Central.
func NewMTLSHTTPClient() (*http.Client, error) {
	caPEM, err := mtls.CACertPEM()
	if err != nil {
		return nil, fmt.Errorf("loading CA certificate: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to add CA certificate to pool")
	}

	cert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("loading client certificate: %w", err)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      certPool,
				Certificates: []tls.Certificate{cert},
			},
		},
		Timeout: 5 * time.Minute,
	}, nil
}
