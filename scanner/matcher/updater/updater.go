package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/matcher/metadata/postgres"
)

const (
	ifModifiedSince = `If-Modified-Since`
	lastModified    = `Last-Modified`

	name = `scanner-v4-updater`

	updateFilePattern = `updates-*.json.zst`
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

type Opts struct {
	Store         datastore.MatcherStore
	Locker        *ctxlock.Locker
	Pool          *pgxpool.Pool
	MetadataStore postgres.MetadataStore

	Client         *http.Client
	URL            string
	Root           string
	UpdateInterval time.Duration

	SkipGC          bool
	UpdateRetention int
}

type Updater struct {
	ctx    context.Context
	cancel context.CancelFunc

	store         datastore.MatcherStore
	locker        updates.LockSource
	pool          *pgxpool.Pool
	metadataStore postgres.MetadataStore

	client         *http.Client
	url            string
	root           string
	updateInterval time.Duration

	skipGC          bool
	updateRetention int

	importVulns func(ctx context.Context, reader io.Reader) error
}

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
	if opts.UpdateRetention <= 1 {
		opts.UpdateRetention = libvuln.DefaultUpdateRetention
	}
	if opts.UpdateInterval < time.Minute {
		opts.UpdateInterval = 5 * time.Minute
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
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/Updater.tryRemoveExiting")

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

// Close cancels the updater's context and prevents future update cycles.
// It does **not** cleanup any other resources. It is the user's responsibility to do so.
func (u *Updater) Close() error {
	u.cancel()
	return nil
}

// Start periodically updates the vulnerability data via Update.
// Each period is adjusted by some amount of jitter.
func (u *Updater) Start() error {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/Updater.Start")

	zlog.Info(ctx).Msg("starting initial update")
	if err := u.update(ctx); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
	}
	zlog.Info(ctx).Msg("completed initial update")

	timer := time.NewTimer(u.updateInterval + jitter())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			zlog.Info(ctx).Msg("starting update")
			if err := u.update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			}
			zlog.Info(ctx).Msg("completed update")

			timer.Reset(u.updateInterval + jitter())
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// update runs the full vulnerability update process.
func (u *Updater) update(ctx context.Context) error {
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
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.runUpdate")

	ctx, done := u.locker.TryLock(ctx, name)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Info(ctx).
			Str("updater", name).
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
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.fetch")

	zlog.Info(ctx).Str("url", u.url).Msg("fetching vuln update")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	req.Header.Set(ifModifiedSince, prevTimestamp.Format(http.TimeFormat))

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotModified:
		// No updates.
		return nil, time.Time{}, nil
	default:
		return nil, time.Time{}, fmt.Errorf("received status code %q querying update endpoint", resp.StatusCode)
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

// runGC runs a garbage collection cycle.
// This is heavily copied from https://github.com/quay/claircore/blob/v1.5.20/libvuln/updates/manager.go#L233.
func (u *Updater) runGC(ctx context.Context) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.runGC")

	ctx, done := u.locker.TryLock(ctx, "garbage-collection")
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection already running")
		return
	}

	zlog.Info(ctx).Int("retention", u.updateRetention).Msg("GC started")
	i, err := u.store.GC(ctx, u.updateRetention)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("performing GC")
		return
	}
	zlog.Info(ctx).
		Int64("remaining_ops", i).
		Int("retention", u.updateRetention).
		Msg("GC completed")
}

func jitter() time.Duration {
	return jitterMinutes[random.Intn(len(jitterMinutes))]
}
