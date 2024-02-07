package vuln

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const (
	ifModifiedSince = `If-Modified-Since`
	lastModified    = `Last-Modified`

	updateName = `scanner-v4-updater`

	updateFilePattern = `updates-*.json.zst`

	defaultUpdateInterval  = 5 * time.Minute
	defaultUpdateRetention = libvuln.DefaultUpdateRetention
	defaultFullGCInterval  = 24 * time.Hour

	defaultRetryMax   = 4
	defaultRetryDelay = 10 * time.Second
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	jitterMinutes = []time.Duration{
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		20 * time.Minute,
	}
)

// Opts represents Updater options.
//
// Store, Locker, Pool, MetadataStore, and URL are required.
// The rest are optional.
type Opts struct {
	Store         datastore.MatcherStore
	Locker        *ctxlock.Locker
	Pool          *pgxpool.Pool
	MetadataStore postgres.MatcherMetadataStore

	Client         *http.Client
	URL            string
	Root           string
	UpdateInterval time.Duration

	SkipGC          bool
	UpdateRetention int
	FullGCInterval  time.Duration

	RetryDelay time.Duration
	RetryMax   int
}

// Updater represents a vulnerability updater.
// An Updater reaches out to a given URL periodically to fetch the latest vulnerability data.
type Updater struct {
	ctx    context.Context
	cancel context.CancelFunc

	store         datastore.MatcherStore
	locker        updates.LockSource
	pool          *pgxpool.Pool
	metadataStore postgres.MatcherMetadataStore

	client         *http.Client
	url            string
	root           string
	updateInterval time.Duration
	initialized    atomic.Bool

	skipGC          bool
	updateRetention int
	fullGCInterval  time.Duration

	importVulns func(ctx context.Context, reader io.Reader) error

	retryDelay time.Duration
	retryMax   int
}

// New creates a new Updater based on the given options.
func New(ctx context.Context, opts Opts) (*Updater, error) {
	if err := fillOpts(&opts); err != nil {
		return nil, fmt.Errorf("invalid updater options: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	u := &Updater{
		ctx:    ctx,
		cancel: cancel,

		store:         opts.Store,
		locker:        opts.Locker,
		pool:          opts.Pool,
		metadataStore: opts.MetadataStore,

		client:         opts.Client,
		url:            opts.URL,
		root:           opts.Root,
		updateInterval: opts.UpdateInterval,

		skipGC:          opts.SkipGC,
		updateRetention: opts.UpdateRetention,
		fullGCInterval:  opts.FullGCInterval,

		retryDelay: opts.RetryDelay,
		retryMax:   opts.RetryMax,
	}
	u.importVulns = func(ctx context.Context, reader io.Reader) error {
		return libvuln.OfflineImport(ctx, u.pool, reader)
	}

	u.tryRemoveExiting()

	return u, nil
}

func fillOpts(opts *Opts) error {
	if opts == nil {
		panic("programmer error")
	}
	if err := validate(*opts); err != nil {
		return err
	}

	if opts.Client == nil {
		opts.Client = http.DefaultClient
	}
	if opts.Root == "" {
		opts.Root = os.TempDir()
	}
	if opts.UpdateInterval < time.Minute {
		opts.UpdateInterval = defaultUpdateInterval
	}
	if opts.UpdateRetention <= 1 {
		opts.UpdateRetention = defaultUpdateRetention
	}
	if opts.FullGCInterval < 3*time.Hour {
		opts.FullGCInterval = defaultFullGCInterval
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = defaultRetryDelay
	}
	if opts.RetryMax == 0 {
		opts.RetryMax = defaultRetryMax
	}
	return nil
}

func validate(opts Opts) error {
	if opts.Store == nil || opts.Locker == nil || opts.Pool == nil || opts.MetadataStore == nil {
		return errors.New("must provide a Store, a Locker, a Pool, and a MetadataStore")
	}
	if _, err := url.Parse(opts.URL); err != nil {
		return fmt.Errorf("invalid URL: %q", opts.URL)
	}
	return nil
}

// tryRemoveExiting attempts to remove any leftover update files present in the updater's
// root directory.
//
// It is assumed the previous updater used the same root directory.
// Any errors when attempting to remove pre-existing files are simply logged.
func (u *Updater) tryRemoveExiting() {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/vuln/Updater.tryRemoveExiting")

	files, err := fs.Glob(os.DirFS(u.root), updateFilePattern)
	if err != nil {
		zlog.Warn(ctx).Err(err).Msg("searching for previous update files to remove")
	}
	for _, file := range files {
		if err := os.Remove(filepath.Join(u.root, file)); err != nil {
			zlog.Warn(ctx).Err(err).Msgf("removing previous update file %s", file)
		}
	}
}

// Stop cancels the updater's context and prevents future update cycles.
// It does **not** cleanup any other resources. It is the user's responsibility to do so.
func (u *Updater) Stop() error {
	u.cancel()
	return nil
}

// Start periodically updates the vulnerability data.
// Each period is adjusted by some amount of jitter.
func (u *Updater) Start() error {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/vuln/Updater.Start")

	if !u.skipGC {
		go u.runGCFullPeriodic()
	}

	timer := time.NewTimer(u.updateInterval + jitter())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			zlog.Info(ctx).Msg("starting update")
			if err := u.Update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			}
			zlog.Info(ctx).Msg("completed update")

			timer.Reset(u.updateInterval + jitter())
		}
	}
}

// Initialized returns true if the vulnerability updater has fully initialized
// the vulnerability store.
func (u *Updater) Initialized(ctx context.Context) bool {
	if u.initialized.Load() {
		return true
	}
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.Initialized")
	ts, err := u.metadataStore.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		zlog.
			Warn(ctx).
			Err(err).
			Msg("did not get previous vuln update timestamp")
		return false
	}
	if ts.IsZero() {
		return false
	}
	zlog.Info(ctx).Msg("previous run exists: setting updater to initialized")
	u.initialized.Store(true)
	return true
}

// Update runs the full vulnerability update process.
//
// Note: periodic full GC will not be started.
func (u *Updater) Update(ctx context.Context) error {
	if err := u.runUpdate(ctx); err != nil {
		return err
	}
	if !u.skipGC {
		u.runGC(ctx)
	}
	return nil
}

// runUpdate updates the vulnerability data.
func (u *Updater) runUpdate(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.runUpdate")

	// Use TryLock instead of Lock to prevent simultaneous updates.
	ctx, done := u.locker.TryLock(ctx, updateName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Info(ctx).
			Str("updater", updateName).
			Msg("did not obtain lock, skipping update run")
		return nil
	}

	prevTimestamp, err := u.metadataStore.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("did not get previous vuln update timestamp")
		return err
	}
	zlog.Info(ctx).
		Str("timestamp", prevTimestamp.Format(http.TimeFormat)).
		Msg("previous vuln update")

	f, timestamp, err := u.fetch(ctx, prevTimestamp)
	if err != nil {
		return err
	}
	if f == nil {
		// Nothing to update at this time.
		zlog.Info(ctx).Msg("no new vulnerability update")
		return nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("removing temp update file %q", f.Name())
		}
	}()

	dec, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating zstd reader: %w", err)
	}
	defer dec.Close()

	if err := u.importVulns(ctx, dec); err != nil {
		return err
	}

	if err := u.metadataStore.SetLastVulnerabilityUpdate(ctx, timestamp); err != nil {
		return err
	}

	if u.initialized.CompareAndSwap(false, true) {
		zlog.Info(ctx).Msg("finished initial updater run: setting updater to initialized")
	}

	zlog.Info(ctx).
		Str("timestamp", timestamp.Format(http.TimeFormat)).
		Msg("new vuln update")

	return nil
}

// fetch acquires the latest vulnerability data and writes it to the returned file.
// If successful, fetch returns the `Last-Modified` timestamp as well.
// If there have been no updates since prevTimestamp, the returned file will be nil.
// It is up to the user to close and delete the returned file.
func (u *Updater) fetch(ctx context.Context, prevTimestamp time.Time) (*os.File, time.Time, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.fetch")

	var resp *http.Response
	var err error
	for attempt := 0; attempt < u.retryMax; attempt++ {
		zlog.Info(ctx).Str("url", u.url).Msg("fetching vuln update")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
		if err != nil {
			return nil, time.Time{}, err
		}
		req.Header.Set(ifModifiedSince, prevTimestamp.Format(http.TimeFormat))

		resp, err = u.client.Do(req)
		if err != nil {
			var opErr *net.OpError
			if errors.As(err, &opErr) && opErr.Op == "dial" && strings.Contains(opErr.Err.Error(), "connection refused") {
				// Retry when vuln URL is unavailable
				zlog.Error(ctx).Err(err).Msg("connection refused, retrying...")
				if attempt < u.retryMax-1 {
					time.Sleep(u.retryDelay) // Wait for retryDelay before retrying
					continue
				}
			}
			// For other errors, do not retry
			closeResponse(resp)
			return nil, time.Time{}, err
		}
		break // no error, no retry
	}
	defer closeResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotModified:
		// No updates.
		return nil, time.Time{}, nil
	default:
		return nil, time.Time{}, fmt.Errorf("received status code %d querying update endpoint", resp.StatusCode)
	}

	f, err := os.CreateTemp(u.root, updateFilePattern)
	if err != nil {
		return nil, time.Time{}, err
	}
	var succeeded bool
	defer func() {
		if succeeded {
			return
		}
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("removing temp update file %q", f.Name())
		}
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("writing vulnerabilities to temp file: %w", err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, time.Time{}, fmt.Errorf("seeking temp update file to start: %w", err)
	}

	modTime := resp.Header.Get(lastModified)
	timestamp, err := http.ParseTime(modTime)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing Last-Modified header: %w", err)
	}
	succeeded = true
	return f, timestamp, nil
}

func jitter() time.Duration {
	return jitterMinutes[random.Intn(len(jitterMinutes))]
}

func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		utils.IgnoreError(resp.Body.Close)
	}
}
