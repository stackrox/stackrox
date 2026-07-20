package cmd

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

const (
	mappingFetchMaxAttempts = 3
	mappingFetchBaseBackoff = 2 * time.Second
	mappingClientTimeout    = 30 * time.Second
	mappingRefreshInterval  = time.Hour
)

// newMappingDownloader builds the filedownloader.Downloader that keeps the
// repository-to-CPE mapping file at cachePath fresh. scan() only reads the
// local file, so a flaky mapping endpoint never blocks a scan directly.
//
// WithHTTPClient overrides filedownloader's default retryablehttp client so
// WithRetryPolicy is the only retry layer (nesting both would multiply
// worst-case latency on the mandatory initial fetch).
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
