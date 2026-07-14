package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/retry"
)

// Retry policy for a single repository-to-CPE mapping file fetch: absorbs
// transient network errors on the mapping HTTP call without waiting a full
// refresh interval - or, for the mandatory initial fetch, failing agent
// startup - over a single blip.
const (
	mappingFetchMaxAttempts = 3
	mappingFetchBaseBackoff = 2 * time.Second

	// mappingClientTimeout bounds a single mapping fetch attempt (the
	// mandatory initial fetch at startup, and each periodic refresh); it
	// does not affect scans, which always read the local cached file.
	// Hardcoded for now; make it a --mapping-client-timeout flag later if
	// reviewers ask for it to be tunable.
	mappingClientTimeout = 30 * time.Second

	// mappingRefreshInterval is how often the mapping file is refreshed,
	// independent of rescan-interval. Hardcoded for now; make it a
	// --mapping-refresh-interval flag later if reviewers ask for it to be
	// tunable.
	mappingRefreshInterval = time.Hour
)

// mappingRefresher periodically fetches the repository-to-CPE mapping file
// over HTTP and atomically publishes it to a local cache file, decoupling
// that network call from every individual scan: scan() only ever reads the
// local cache file (via Repo2CPEMappingFile), so a slow or flaky mapping
// endpoint can never block or fail a scan directly - only fetchOnce's
// mandatory, synchronous initial call (which gates the first scan, see
// runServe) can.
type mappingRefresher struct {
	url       string
	cachePath string
	client    *http.Client

	// fetchFn defaults to m.fetch; tests override it to inject failures
	// without a real HTTP server. newTick defaults to time.After; tests
	// substitute a function returning a manually driven channel for
	// precise control over Run's loop.
	fetchFn func(ctx context.Context) error
	newTick func(d time.Duration) <-chan time.Time
}

func newMappingRefresher(url, cachePath string) *mappingRefresher {
	m := &mappingRefresher{
		url:       url,
		cachePath: cachePath,
		client:    &http.Client{Transport: proxy.RoundTripper()},
		newTick:   time.After,
	}
	m.fetchFn = m.fetch
	return m
}

// fetchOnce downloads the mapping file and atomically replaces cachePath's
// contents, retrying a bounded number of times with backoff to ride out
// transient errors. Called synchronously once at startup - where its
// result gates the initial scan - and then periodically by Run.
//
// The backoff wait is done in BetweenAttempts, via a ctx-aware select,
// rather than via retry.WithExponentialBackoff's plain time.Sleep, so a
// cancellation during the wait is honored immediately instead of only
// after the full backoff elapses. One consequence: if ctx is cancelled
// during that wait, retry.WithRetry still makes one more (fast, guaranteed
// to fail) fetchFn call before its own next-iteration check catches the
// cancellation - retry.WithRetry has no hook to skip a retry already in
// flight when cancellation is observed between attempts.
func (m *mappingRefresher) fetchOnce(ctx context.Context) error {
	attempt := 0
	backoff := mappingFetchBaseBackoff
	err := retry.WithRetry(func() error { return m.fetchFn(ctx) },
		retry.Tries(mappingFetchMaxAttempts),
		retry.WithContext(ctx),
		retry.OnFailedAttempts(func(err error) {
			attempt++
			log.Warnf("Mapping file fetch attempt %d/%d failed: %v; retrying in %v",
				attempt, mappingFetchMaxAttempts, err, backoff)
		}),
		retry.BetweenAttempts(func(int) {
			select {
			case <-ctx.Done():
			case <-time.After(backoff):
			}
			backoff *= 2
		}),
	)
	if err != nil {
		return fmt.Errorf("fetching mapping file after %d attempts: %w", mappingFetchMaxAttempts, err)
	}
	return nil
}

// fetch performs a single fetch-and-publish attempt: downloads m.url and
// atomically replaces cachePath's contents via a temp file plus rename, so
// a concurrent scan reading the old file never observes a partial write.
func (m *mappingRefresher) fetch(ctx context.Context) error {
	fetchCtx, cancel := context.WithTimeout(ctx, mappingClientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, m.url, nil)
	if err != nil {
		return fmt.Errorf("building request for %q: %w", m.url, err)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %q: %w", m.url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %q: unexpected status %s", m.url, resp.Status)
	}

	dir := filepath.Dir(m.cachePath)
	tmp, err := os.CreateTemp(dir, filepath.Base(m.cachePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file in %q: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once renamed below

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing %q: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, m.cachePath); err != nil {
		return fmt.Errorf("publishing %q: %w", m.cachePath, err)
	}
	log.Infof("Repository-to-CPE mapping file refreshed from %s", m.url)
	return nil
}

// Run refreshes the mapping file every mappingRefreshInterval. Failures are
// logged, not propagated: scan() keeps consulting the last successfully
// fetched file already published at cachePath, so a failure here only means
// the mapping data goes one interval longer without an update, never that
// scanning stops working. Blocks until ctx is cancelled.
func (m *mappingRefresher) Run(ctx context.Context) {
	tick := m.newTick(mappingRefreshInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			if err := m.fetchOnce(ctx); err != nil {
				log.Warnf("Periodic mapping file refresh failed (harmless: scans keep using the "+
					"last successfully fetched file): %v", err)
			}
			tick = m.newTick(mappingRefreshInterval)
		}
	}
}
