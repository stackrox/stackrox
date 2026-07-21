package cmd

import (
	"time"

	"github.com/stackrox/rox/pkg/filedownloader"
)

const (
	mappingFetchRetryMax     = 3 // 1 initial attempt + 3 retries = 4 total attempts against a flaky endpoint.
	mappingFetchRetryWaitMin = 5 * time.Second
	mappingClientTimeout     = 40 * time.Second
	mappingRefreshInterval   = time.Hour
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
