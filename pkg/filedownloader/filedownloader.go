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
	defaultRetryMax       = 4
	defaultRetryWaitMin   = 10 * time.Second
)

var log = logging.LoggerForModule()

// Option configures a Downloader.
type Option func(*Downloader)

// WithHTTPClient overrides the default HTTP client.
// Bypasses WithRetryMax / WithRetryWaitMin.
func WithHTTPClient(c *http.Client) Option {
	return func(d *Downloader) { d.client = c }
}

// WithMaxSize overrides the default maximum response body size (5 MB).
func WithMaxSize(n int64) Option {
	return func(d *Downloader) { d.maxSize = n }
}

// WithRequestTimeout overrides the default per-DownloadOnce timeout (60s).
// The timeout covers the entire call, including any retryablehttp retries.
func WithRequestTimeout(t time.Duration) Option {
	return func(d *Downloader) { d.requestTimeout = t }
}

// WithOnComplete sets a callback invoked once per DownloadOnce call (not once
// per retry attempt), with the cumulative duration of that call.
func WithOnComplete(fn func(err error, duration time.Duration)) Option {
	return func(d *Downloader) { d.onComplete = fn }
}

// WithRetryMax sets retryablehttp RetryMax (default 4). Ignored with WithHTTPClient.
func WithRetryMax(n int) Option {
	return func(d *Downloader) { d.retryMax = n }
}

// WithRetryWaitMin sets retryablehttp RetryWaitMin (default 10s). Ignored with WithHTTPClient.
func WithRetryWaitMin(d time.Duration) Option {
	return func(dl *Downloader) { dl.retryWaitMin = d }
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
	retryMax       int
	retryWaitMin   time.Duration
	stopSig        concurrency.Signal
	doneSig        concurrency.Signal
}

// New creates a Downloader that periodically fetches url and writes the response to filePath.
func New(url, filePath string, interval time.Duration, opts ...Option) *Downloader {
	if interval < minInterval {
		log.Warnf("Download interval %v is below minimum %v, clamping", interval, minInterval)
		interval = minInterval
	}

	d := &Downloader{
		url:            url,
		filePath:       filePath,
		interval:       interval,
		maxSize:        defaultMaxSize,
		requestTimeout: defaultRequestTimeout,
		retryMax:       defaultRetryMax,
		retryWaitMin:   defaultRetryWaitMin,
		stopSig:        concurrency.NewSignal(),
		doneSig:        concurrency.NewSignal(),
	}
	for _, o := range opts {
		o(d)
	}

	if d.client == nil {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = d.retryMax
		retryClient.RetryWaitMin = d.retryWaitMin
		retryClient.Logger = pkgRetryableHTTP.NewDebugLogger(log)
		retryClient.HTTPClient.Transport = proxy.RoundTripper()
		d.client = retryClient.StandardClient()
	}
	return d
}

// Start downloads filePath, then refreshes it periodically in the
// background. waitForInitial blocks until the first download succeeds.
func (d *Downloader) Start(ctx context.Context, waitForInitial bool) error {
	log.Infof("Starting file downloader for %q → %q", d.url, d.filePath)

	// Fail fast on a persistent directory problem before downloading.
	if err := d.ensureDestinationDir(); err != nil {
		d.reportRunFailure(err)
		d.doneSig.Signal() // no goroutine will run; Stop must not block on it
		return err
	}

	if waitForInitial {
		if err := d.DownloadOnce(ctx); err != nil {
			d.doneSig.Signal() // no goroutine will run; Stop must not block on it
			return err
		}
	}

	go func() {
		defer d.doneSig.Signal()
		loopCtx, cancel := concurrency.DependentContext(ctx, &d.stopSig)
		defer cancel()
		if !waitForInitial {
			_ = d.DownloadOnce(loopCtx)
		}
		d.tickLoop(loopCtx)
	}()
	return nil
}

// Stop signals the downloader to stop and blocks until it exits.
func (d *Downloader) Stop() {
	d.stopSig.Signal()
	<-d.doneSig.Done()
}

func (d *Downloader) ensureDestinationDir() error {
	if err := os.MkdirAll(filepath.Dir(d.filePath), 0700); err != nil {
		return fmt.Errorf("creating directory for %q: %w", d.filePath, err)
	}
	return nil
}

// reportRunFailure reports a fatal setup error via OnComplete if
// configured, otherwise a log line - same as DownloadOnce does for errors.
func (d *Downloader) reportRunFailure(err error) {
	if d.onComplete != nil {
		d.onComplete(err, 0)
	} else {
		log.Errorf("Downloader will not run: %v", err)
	}
}

// tickLoop is the periodic download loop behind Start.
func (d *Downloader) tickLoop(ctx context.Context) {
	t := time.NewTimer(d.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			_ = d.DownloadOnce(ctx)
			t.Reset(d.interval)
		case <-ctx.Done():
			return
		}
	}
}

// DownloadOnce performs one fetch-and-atomic-write cycle. Retries for
// transient HTTP failures are handled by the underlying retryablehttp
// client. Invokes the OnComplete callback once with the cumulative
// duration; logs on failure if no callback is set.
func (d *Downloader) DownloadOnce(ctx context.Context) error {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()
	err := d.doDownload(ctx)
	duration := time.Since(start)
	if d.onComplete != nil {
		d.onComplete(err, duration)
	} else if err != nil {
		log.Warnf("Download of %q failed: %v", d.url, err)
	}
	return err
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
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}
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
