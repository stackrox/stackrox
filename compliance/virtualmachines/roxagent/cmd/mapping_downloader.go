package cmd

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/httputil/proxy"
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

// newMappingDownloader builds the filedownloader.Downloader responsible for
// keeping the repository-to-CPE mapping file at cachePath fresh, decoupling
// that network call from every individual scan: scan() only ever reads the
// local cache file (via Repo2CPEMappingFile), so a slow or flaky mapping
// endpoint can never block or fail a scan directly - only a mandatory,
// synchronous DownloadOnce call (which gates the first scan, see runServe)
// can.
//
// The HTTP client has no retries of its own (WithHTTPClient overrides
// filedownloader's default retryablehttp client), so WithRetryPolicy's
// attempts/backoff remain the only retry layer: nesting both would multiply
// worst-case latency for the mandatory initial fetch well beyond what
// "fail fast on a persistently broken mapping endpoint" requires.
func newMappingDownloader(url, cachePath string) *filedownloader.Downloader {
	return filedownloader.New(url, cachePath, mappingRefreshInterval,
		filedownloader.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
		filedownloader.WithRequestTimeout(mappingClientTimeout),
		filedownloader.WithRetryPolicy(mappingFetchMaxAttempts, mappingFetchBaseBackoff),
		filedownloader.WithOnComplete(func(err error, _ time.Duration) {
			if err != nil {
				log.Infof("Mapping file refresh failed (scans keep using the last successfully fetched file): %v", err)
			} else {
				log.Infof("Repository-to-CPE mapping file refreshed from %s", url)
			}
		}),
	)
}
