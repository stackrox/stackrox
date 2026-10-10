package index

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/filedownloader"
	pkgRetryableHTTP "github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
)

const (
	// mappingDownloadInterval is unused by DownloadOnce. New still requires
	// one, and the downloader clamps anything under five minutes.
	mappingDownloadInterval = time.Hour
	// mappingDownloadStagingSuffix keeps the download off the destination
	// until validation passes, so a bad body cannot replace a good file.
	mappingDownloadStagingSuffix = ".download"

	// Six attempts (initial plus five retries). WaitMin 5s doubles up to
	// WaitMax and still finishes inside mappingDownloadTimeout.
	mappingDownloadRetryMax     = 5
	mappingDownloadRetryWaitMin = 5 * time.Second
	mappingDownloadRetryWaitMax = 30 * time.Second
	mappingDownloadTimeout      = 2 * time.Minute
)

// mappingConfig downloads the repo-to-CPE mapping when file download is
// enabled. The URL is cleared afterward because Claircore fetches any
// non-empty URL, and a failed download returns before the host is opened.
func (l *localNodeIndexer) mappingConfig(ctx context.Context) (NodeIndexerConfig, error) {
	cfg := l.cfg
	if !env.NodeIndexMappingFileDownload.BooleanSetting() {
		return cfg, nil
	}
	if cfg.Repo2CPEMappingURL == "" {
		if cfg.Repo2CPEMappingFile == "" {
			return cfg, errors.New("repo-to-CPE mapping URL is empty")
		}
		return cfg, nil
	}

	path := env.NodeIndexMappingFile.Setting()
	// Join(path, suffix) would treat ".download" as a child directory.
	stagingPath := filepath.Join(filepath.Dir(path), filepath.Base(path)+mappingDownloadStagingSuffix)
	client, err := mappingDownloadClient(cfg)
	if err != nil {
		return cfg, err
	}
	downloader := filedownloader.New(cfg.Repo2CPEMappingURL, stagingPath, mappingDownloadInterval,
		filedownloader.WithHTTPClient(client),
		filedownloader.WithRequestTimeout(mappingDownloadTimeout),
		filedownloader.WithMaxSize(cpemapping.MaxMappingBytes),
	)
	if err := downloader.DownloadOnce(ctx); err != nil {
		return cfg, errors.Wrap(err, "downloading repo-to-CPE mapping")
	}
	content, err := os.ReadFile(stagingPath)
	if err != nil {
		return cfg, errors.Wrap(err, "reading downloaded repo-to-CPE mapping")
	}
	if err := cpemapping.ValidateMapping(content); err != nil {
		return cfg, errors.Wrap(err, "validating repo-to-CPE mapping")
	}
	if err := filedownloader.AtomicWriteFile(path, content); err != nil {
		return cfg, errors.Wrap(err, "publishing repo-to-CPE mapping")
	}
	log.Infof("Downloaded repo-to-CPE mapping from %q to %q (%d bytes)", cfg.Repo2CPEMappingURL, path, len(content))
	cfg.Repo2CPEMappingURL = ""
	cfg.Repo2CPEMappingFile = path
	return cfg, nil
}

// mappingDownloadClient retries on the Sensor mTLS transport. The file
// downloader's default client has no client certificate, and WithHTTPClient
// does not apply the downloader's own retry settings.
func mappingDownloadClient(cfg NodeIndexerConfig) (*http.Client, error) {
	base := cfg.Client
	if base == nil {
		var err error
		base, err = newDefaultClient(cfg.Repo2CPEMappingURL)
		if err != nil {
			return nil, errors.Wrap(err, "creating repo-to-CPE mapping HTTP client")
		}
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = mappingDownloadRetryMax
	retryClient.RetryWaitMin = mappingDownloadRetryWaitMin
	retryClient.RetryWaitMax = mappingDownloadRetryWaitMax
	retryClient.Logger = pkgRetryableHTTP.NewDebugLogger(log)
	retryClient.HTTPClient = &http.Client{Transport: base.Transport}
	return retryClient.StandardClient(), nil
}
