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
	"github.com/stackrox/rox/pkg/retry"
	pkgRetryableHTTP "github.com/stackrox/rox/pkg/retryablehttp"
)

const (
	defaultMaxSize          = 5 * 1024 * 1024 // 5 MB
	defaultRequestTimeout   = 60 * time.Second
	minInterval             = 5 * time.Minute
	defaultRetryMaxAttempts = 1
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

// WithOnComplete sets a callback invoked once per DownloadOnce call (not once
// per retry attempt), with the cumulative duration of that call.
func WithOnComplete(fn func(err error, duration time.Duration)) Option {
	return func(d *Downloader) { d.onComplete = fn }
}

// WithRetryPolicy configures DownloadOnce to retry a failed attempt up to
// maxAttempts total attempts (including the first), waiting baseBackoff
// before the first retry and doubling the wait after each subsequent
// failure. The wait is ctx-aware: cancellation during it aborts immediately
// rather than completing the full backoff.
//
// Defaults to maxAttempts=1, i.e. no added retry layer beyond whatever
// http.Client is configured (WithHTTPClient). Combining a retrying
// http.Client (the default) with maxAttempts > 1 nests two retry policies
// and can multiply worst-case latency; callers that want WithRetryPolicy to
// be the only retry layer should also pass a non-retrying WithHTTPClient.
//
// Panics if maxAttempts < 1 or baseBackoff < 0. Every call site in this
// codebase passes compile-time literals, so an invalid value is a
// programmer error that should fail loudly at startup, not a runtime
// condition to clamp or degrade gracefully from.
func WithRetryPolicy(maxAttempts int, baseBackoff time.Duration) Option {
	if maxAttempts < 1 {
		panic(fmt.Sprintf("filedownloader: WithRetryPolicy: maxAttempts must be >= 1, got %d", maxAttempts))
	}
	if baseBackoff < 0 {
		panic(fmt.Sprintf("filedownloader: WithRetryPolicy: baseBackoff must be >= 0, got %v", baseBackoff))
	}
	return func(d *Downloader) {
		d.retryMaxAttempts = maxAttempts
		d.retryBaseBackoff = baseBackoff
	}
}

// Downloader periodically downloads a URL to a local file using atomic writes.
type Downloader struct {
	url              string
	filePath         string
	interval         time.Duration
	client           *http.Client
	maxSize          int64
	requestTimeout   time.Duration
	onComplete       func(err error, duration time.Duration)
	retryMaxAttempts int
	retryBaseBackoff time.Duration
	stopSig          concurrency.Signal
	doneSig          concurrency.Signal
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
		url:              url,
		filePath:         filePath,
		interval:         interval,
		client:           retryClient.StandardClient(),
		maxSize:          defaultMaxSize,
		requestTimeout:   defaultRequestTimeout,
		retryMaxAttempts: defaultRetryMaxAttempts,
		stopSig:          concurrency.NewSignal(),
		doneSig:          concurrency.NewSignal(),
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

	// Checked upfront, in addition to atomicWriteFile's own MkdirAll: a
	// persistent directory problem then aborts the whole loop immediately
	// via onComplete, before ever attempting a network request, instead of
	// only surfacing once the first tick's write fails after an already
	// completed (and wasted) HTTP round trip.
	if err := os.MkdirAll(filepath.Dir(d.filePath), 0700); err != nil {
		mkdirErr := fmt.Errorf("creating directory for %q: %w", d.filePath, err)
		log.Errorf("Downloader will not run: %v", mkdirErr)
		if d.onComplete != nil {
			d.onComplete(mkdirErr, 0)
		}
		return
	}

	_ = d.DownloadOnce(ctx)
	d.tickLoop(ctx)
}

// Run blocks, downloading roughly every interval, until ctx is done, then
// returns ctx.Err(). Each wait is measured from the end of the previous
// download attempt, not a fixed wall-clock cadence, so a slow or retried
// download pushes the next attempt later by its own duration instead of
// overlapping with it. Unlike Start, Run performs no download on entry:
// pair Run with an explicit DownloadOnce call for a mandatory, blocking
// first fetch (e.g. to gate a caller's own startup on it) before starting
// the periodic loop.
//
// Run is an alternative to Start/Stop for callers whose lifecycle is
// already expressed as a single ctx (e.g. one goroutine among several
// tracked by an errgroup); don't mix Run with Start/Stop on the same
// Downloader.
func (d *Downloader) Run(ctx context.Context) error {
	d.tickLoop(ctx)
	return ctx.Err()
}

// tickLoop downloads once per interval until ctx is done, timing each wait
// from the end of the previous download rather than a fixed schedule (see
// Run's doc comment). It is the shared periodic-download loop behind both
// Start (via run) and Run.
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

// DownloadOnce performs a single logical download: it fetches url and
// atomically publishes the result to filePath, retrying according to the
// configured WithRetryPolicy (a single attempt by default), and returns the
// outcome. WithOnComplete's callback, if configured, is invoked once for
// the whole operation (not once per retry attempt) with the cumulative
// duration; otherwise a failure is logged.
//
// Callers needing a mandatory, error-returning first fetch - e.g. to gate
// their own startup on it - call DownloadOnce directly before starting the
// periodic loop via Start or Run.
func (d *Downloader) DownloadOnce(ctx context.Context) error {
	start := time.Now()
	err := d.downloadWithRetry(ctx)
	duration := time.Since(start)
	if d.onComplete != nil {
		d.onComplete(err, duration)
	} else if err != nil {
		log.Warnf("Download of %q failed: %v", d.url, err)
	}
	return err
}

// downloadWithRetry runs downloadAttempt under the configured retry policy:
// a single attempt by default, or up to retryMaxAttempts attempts with
// exponential backoff starting at retryBaseBackoff between failures. With
// no retries configured (the default), downloadAttempt's error is returned
// unwrapped. With retries configured, an error means every attempt failed,
// so it's wrapped with the attempt count - letting callers and logs tell
// "retries exhausted" apart from "the one and only attempt failed".
func (d *Downloader) downloadWithRetry(ctx context.Context) error {
	if d.retryMaxAttempts <= 1 {
		return d.downloadAttempt(ctx)
	}

	attempt := 0
	backoff := d.retryBaseBackoff
	err := retry.WithRetry(func() error { return d.downloadAttempt(ctx) },
		retry.Tries(d.retryMaxAttempts),
		retry.WithContext(ctx),
		retry.OnFailedAttempts(func(err error) {
			attempt++
			log.Warnf("Download of %q failed (attempt %d/%d): %v; retrying in %v",
				d.url, attempt, d.retryMaxAttempts, err, backoff)
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
		return fmt.Errorf("after %d attempts: %w", d.retryMaxAttempts, err)
	}
	return nil
}

// downloadAttempt bounds a single doDownload call by d.requestTimeout.
func (d *Downloader) downloadAttempt(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, d.requestTimeout)
	defer cancel()
	return d.doDownload(ctx)
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
