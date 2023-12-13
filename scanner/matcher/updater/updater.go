package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/matcher/metadata/postgres"
)

const (
	ifModifiedSince = `If-Modified-Since`
	lastModified    = `Last-Modified`

	name = `scanner-v4-updater`
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
	MetadataStore *postgres.MetadataStore

	Client         *http.Client
	URL            string
	Root           string
	UpdateInterval time.Duration

	UpdateRetention int
}

type Updater struct {
	store         datastore.MatcherStore
	locker        *ctxlock.Locker
	pool          *pgxpool.Pool
	metadataStore *postgres.MetadataStore

	client         *http.Client
	url            string
	root           string
	updateInterval time.Duration

	updateRetention int
}

func New(opts Opts) (*Updater, error) {
	if err := fillOpts(&opts); err != nil {
		return nil, fmt.Errorf("invalid updater options: %w", err)
	}

	return &Updater{
		store:         opts.Store,
		locker:        opts.Locker,
		pool:          opts.Pool,
		metadataStore: opts.MetadataStore,

		client:         opts.Client,
		url:            opts.URL,
		root:           opts.Root,
		updateInterval: opts.UpdateInterval,

		updateRetention: opts.UpdateRetention,
	}, nil
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

// Start periodically updates the vulnerability data via Update.
// Each period is adjusted by some amount of jitter.
func (u *Updater) Start(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.Start")

	timer := time.NewTimer(u.updateInterval + jitter())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			zlog.Info(ctx).Msg("starting update")
			if err := u.Update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			}
			zlog.Info(ctx).Msg("completed update")

			timer.Reset(u.updateInterval + jitter())
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Update runs a vulnerability data update, once.
func (u *Updater) Update(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.Update")

	ctx, done := u.locker.TryLock(ctx, name)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Str("updater", name).
			Msg("lock context canceled, excluding from run")
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
		zlog.Info(ctx).Msg("no new vulnerability update")
		// Nothing to update at this time.
		return nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("error closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("error removing temp update file %q", f.Name())
		}
	}()

	dec, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating zstd reader: %w", err)
	}
	defer dec.Close()

	if err := libvuln.OfflineImport(ctx, u.pool, dec); err != nil {
		return err
	}

	if err := u.metadataStore.SetLastVulnerabilityUpdate(ctx, timestamp); err != nil {
		return err
	}
	zlog.Info(ctx).
		Str("timestamp", timestamp).
		Msg("new vuln update")

	// Run garbage collection in the background.
	go u.runGC(ctx)

	return nil
}

// fetch acquires the latest vulnerability data and writes it to the returned file.
// If successful, fetch returns the `Last-Modified` timestamp as well.
// If there have been no updates since the last query, the returned file will be nil.
// It is up to the user to close and delete the returned file.
func (u *Updater) fetch(ctx context.Context, timestamp time.Time) (*os.File, string, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/Updater.fetch")

	zlog.Info(ctx).Str("url", u.url).Msg("fetching vuln update")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set(ifModifiedSince, timestamp.Format(http.TimeFormat))

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotModified:
		// No updates.
		return nil, "", nil
	default:
		return nil, "", fmt.Errorf("received status code %q querying update endpoint", resp.StatusCode)
	}

	f, err := os.CreateTemp(u.root, "updates-*.json.zst")
	if err != nil {
		return nil, "", err
	}
	var succeeded bool
	defer func() {
		if succeeded {
			return
		}
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("error closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("error removing temp update file %q", f.Name())
		}
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("writing vulnerabilities to temp file: %w", err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, "", fmt.Errorf("seeking temp update file to start: %w", err)
	}

	modTime := resp.Header.Get(lastModified)
	succeeded = true
	return f, modTime, nil
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
		zlog.Error(ctx).Err(err).Msg("error while performing GC")
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
