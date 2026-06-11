package filedownloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	pkgRetryableHTTP "github.com/stackrox/rox/pkg/retryablehttp"
)

const (
	defaultMaxSize        = 5 * 1024 * 1024 // 5 MB
	defaultRequestTimeout = 60 * time.Second
	minInterval           = 5 * time.Minute
)

var log = logging.LoggerForModule()

// Option configures a Downloader.
type Option func(*Downloader)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(d *Downloader) { d.client = c }
}

// WithMaxSize overrides the default maximum response body size (5 MB).
func WithMaxSize(n int64) Option {
	return func(d *Downloader) { d.maxSize = n }
}

// WithRequestTimeout overrides the default per-request timeout (60s).
func WithRequestTimeout(t time.Duration) Option {
	return func(d *Downloader) { d.requestTimeout = t }
}

// WithOnComplete sets a callback invoked after each download attempt.
func WithOnComplete(fn func(err error, duration time.Duration)) Option {
	return func(d *Downloader) { d.onComplete = fn }
}

// Downloader periodically downloads a URL to a local file using atomic writes.
type Downloader struct {
	url            string
	filePath       string
	interval       time.Duration
	client         *http.Client
	maxSize        int64
	requestTimeout time.Duration
	onComplete     func(err error, duration time.Duration)
	stopSig        concurrency.Signal
	doneSig        concurrency.Signal
}

// New creates a Downloader that periodically fetches url and writes the response to filePath.
func New(url, filePath string, interval time.Duration, opts ...Option) *Downloader {
	if interval < minInterval {
		log.Warnf("Download interval %v is below minimum %v, clamping", interval, minInterval)
		interval = minInterval
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMin = 10 * time.Second
	retryClient.Logger = pkgRetryableHTTP.NewDebugLogger(log)
	retryClient.HTTPClient.Transport = proxy.RoundTripper()

	d := &Downloader{
		url:            url,
		filePath:       filePath,
		interval:       interval,
		client:         retryClient.StandardClient(),
		maxSize:        defaultMaxSize,
		requestTimeout: defaultRequestTimeout,
		stopSig:        concurrency.NewSignal(),
		doneSig:        concurrency.NewSignal(),
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Start begins periodic downloading in a background goroutine.
func (d *Downloader) Start() {
	log.Infof("Starting file downloader for %q → %q", d.url, d.filePath)
	go d.run()
}

// Stop signals the downloader to stop and blocks until it exits.
func (d *Downloader) Stop() {
	d.stopSig.Signal()
	<-d.doneSig.Done()
}

func (d *Downloader) run() {
	defer d.doneSig.Signal()

	ctx, cancel := concurrency.DependentContext(context.Background(), &d.stopSig)
	defer cancel()

	if err := os.MkdirAll(filepath.Dir(d.filePath), 0700); err != nil {
		mkdirErr := fmt.Errorf("creating directory for %q: %w", d.filePath, err)
		log.Errorf("Downloader will not run: %v", mkdirErr)
		if d.onComplete != nil {
			d.onComplete(mkdirErr, 0)
		}
		return
	}

	d.download(ctx)

	t := time.NewTimer(d.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			d.download(ctx)
			t.Reset(d.interval)
		case <-d.stopSig.Done():
			return
		}
	}
}

func (d *Downloader) download(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()

	start := time.Now()
	err := d.doDownload(ctx)
	duration := time.Since(start)
	if d.onComplete != nil {
		d.onComplete(err, duration)
	} else if err != nil {
		log.Warnf("Download of %q failed: %v", d.url, err)
	}
}

// doDownload performs a single download attempt with atomic file write.
func (d *Downloader) doDownload(ctx context.Context) error {
	log.Debugf("Downloading %q", d.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, nil)
	if err != nil {
		return fmt.Errorf("constructing request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, d.maxSize+1))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}
	if int64(len(body)) > d.maxSize {
		return fmt.Errorf("response body exceeds maximum size of %d bytes", d.maxSize)
	}

	if err := atomicWriteFile(d.filePath, body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	log.Debugf("Successfully downloaded %q → %q", d.url, d.filePath)
	return nil
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".download-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("setting temp file permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
