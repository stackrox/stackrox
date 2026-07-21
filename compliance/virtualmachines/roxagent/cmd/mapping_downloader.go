package cmd

import (
	"time"

	"github.com/stackrox/rox/pkg/filedownloader"
)

const (
	mappingFetchRetryMax     = 3 // RetryMax: 1 initial + 3 retries = 4 attempts.
	mappingFetchRetryWaitMin = 5 * time.Second
	// Must cover retryablehttp backoff plus request time. With WaitMin=5s and
	// retryablehttp's default WaitMax=30s, waits are 5+10+20=35s; leave
	// headroom for four HTTP attempts on a multi-MB mapping file.
	mappingClientTimeout   = 2 * time.Minute
	mappingRefreshInterval = time.Hour
)

// newMappingDownloader builds the filedownloader.Downloader that keeps the
// repository-to-CPE mapping file at cachePath fresh. scan() only reads the
// local file, so a flaky mapping endpoint never blocks a scan directly.
func newMappingDownloader(url, cachePath string) *filedownloader.Downloader {
	return filedownloader.New(url, cachePath, mappingRefreshInterval,
		filedownloader.WithRequestTimeout(mappingClientTimeout),
		filedownloader.WithRetryMax(mappingFetchRetryMax),
		filedownloader.WithRetryWaitMin(mappingFetchRetryWaitMin),
		filedownloader.WithOnComplete(func(err error, _ time.Duration) {
			if err != nil {
				log.Infof("Mapping file refresh failed (scans keep using the last successfully fetched file): %v", err)
			} else {
				log.Infof("Repository-to-CPE mapping file refreshed from %s", url)
			}
		}),
	)
}
